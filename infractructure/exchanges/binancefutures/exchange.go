package binancefutures

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/H3Cki/go-binance/v2/futures"
	"go.uber.org/zap"
)

var (
	eiFileName = "binancefutures_ei.json"
	maxEiAge   = 24 * time.Hour
)

func eiFn() string {
	if futures.UseTestnet {
		return "testnet_" + eiFileName
	}
	return eiFileName
}

type UserConfig struct {
	Testnet    bool   `json:"testnet"`
	API_KEY    string `json:"API_KEY" validate:"required"`
	SECRET_KEY string `json:"SECRET_KEY" validate:"required"`
}

type Config struct {
	ExchangeInfoer outbound.ExchangeInfoer[ExchangeInfo]
	UserConfig     UserConfig
}

type Exchange struct {
	logger *zap.SugaredLogger
	client *futures.Client
	ei     ExchangeInfo
	eier   outbound.ExchangeInfoer[ExchangeInfo]
}

type ExchangeInfo futures.ExchangeInfo

type orderIdentification struct {
	OrderID int64  `json:"orderID"`
	Symbol  string `json:"symbol"`
}

func New(logger *zap.SugaredLogger, cfg Config) *Exchange {
	futures.UseTestnet = cfg.UserConfig.Testnet
	return &Exchange{
		logger: logger,
		client: futures.NewClient(cfg.UserConfig.API_KEY, cfg.UserConfig.SECRET_KEY),
		eier:   cfg.ExchangeInfoer,
	}
}

func (f *Exchange) Init(ctx context.Context) error {
	if err := f.client.NewPingService().Do(ctx); err != nil {
		return err
	}
	_, err := f.info(ctx, false)
	return err
}

// info tries to read the ei from file, if it doesn't exist or is outdated it attempts to fetch the ei
func (f *Exchange) info(ctx context.Context, force bool) (updated bool, err error) {
	// Load if not exists
	ei, err := f.eier.Read(eiFn())
	if force || os.IsNotExist(err) {
		ei, err = f.getExchangeInfo(ctx)
		if err != nil {
			return false, err
		}

		f.ei = ei

		// Ignore save error
		if err := f.eier.Save(eiFn(), ei); err != nil {
			f.logger.Errorf("error saving exchange info: %v", err)
		}
		return true, nil
	}

	f.ei = ei

	// Try to fetch the ei if it's outdated
	if time.Since(time.Unix(ei.ServerTime, 0)) > maxEiAge {
		ei, err = f.getExchangeInfo(ctx)
		if err != nil {
			f.logger.Errorf("error saving exchange info: %v", err)
			return false, err
		}

		f.ei = ei

		// Ignore save error
		if err := f.eier.Save(eiFn(), ei); err != nil {
			f.logger.Errorf("error saving exchange info: %v", err)
		}
	}

	return true, nil
}

func (f *Exchange) getExchangeInfo(ctx context.Context) (ExchangeInfo, error) {
	ei, err := f.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return ExchangeInfo{}, err
	}
	return ExchangeInfo(*ei), err
}

func (f *Exchange) symbol(ctx context.Context, symbol string) (futures.Symbol, error) {
	eiUpdated, err := f.info(ctx, false)
	if err != nil {
		return futures.Symbol{}, err
	}

	for _, fsymbol := range f.ei.Symbols {
		if fsymbol.Symbol == symbol {
			return fsymbol, nil
		}
	}

	// ExchangeInfo was fresh yet such symbol was not found
	if eiUpdated {
		return futures.Symbol{}, fmt.Errorf("unknown symbol: %s", symbol)
	}

	// ExchangeInfo was not fresh, force reload and try finding symbol again
	eiUpdated, err = f.info(ctx, true)
	if err != nil {
		return futures.Symbol{}, err
	}

	for _, fsymbol := range f.ei.Symbols {
		if fsymbol.Symbol == symbol {
			return fsymbol, nil
		}
	}

	return futures.Symbol{}, fmt.Errorf("unknown symbol: %s", symbol)
}

func (f *Exchange) GetPrice(ctx context.Context, req outbound.GetPriceRequest) (float64, error) {
	klinesSvc := f.client.NewKlinesService()
	klines, err := klinesSvc.Symbol(pairToSymbol(req.Pair)).Interval("1m").Limit(1).Do(ctx)
	if err != nil {
		return 0, err
	}

	if len(klines) == 0 {
		return 0, errors.New("no klines returned")
	}

	price, err := strconv.ParseFloat(klines[0].Close, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

type createOrderRequest struct {
	symbol        futures.Symbol
	side          futures.SideType
	orderType     futures.OrderType
	price         float64
	quoteQuantity float64
	baseQuantity  float64
	timeInForce   futures.TimeInForceType
}

func (f *Exchange) CreateOrders(ctx context.Context, req outbound.CreateOrdersRequest) (outbound.ExchangeOrders, error) {
	symbol, err := f.symbol(ctx, pairToSymbol(req.Pair))
	if err != nil {
		return outbound.ExchangeOrders{}, err
	}

	orderSvc, err := corToOrderService(f.client, req, symbol)
	if err != nil {
		return outbound.ExchangeOrders{}, err
	}
	takeProfitSvcs, err := corToTakeProfitServices(f.client, req, symbol)
	if err != nil {
		return outbound.ExchangeOrders{}, err
	}
	stopLossSvcs, err := corToStopLossServices(f.client, req, symbol)
	if err != nil {
		return outbound.ExchangeOrders{}, err
	}

	orderBatch := []*futures.CreateOrderService{orderSvc}
	orderBatch = append(orderBatch, takeProfitSvcs...)
	orderBatch = append(orderBatch, stopLossSvcs...)

	batchRes, err := f.client.NewCreateBatchOrdersService().OrderList(orderBatch).Do(ctx)
	if err != nil {
		for _, order := range batchRes.Orders {
			if err := f.cancelOrder(ctx, order.Symbol, order.OrderID); err != nil {
				f.logger.Errorf("error canceling order: %v", err)
			}
		}

		return outbound.ExchangeOrders{}, err
	}

	f.logger.Info(batchRes)

	errs := []error{}
	//exOrder := newEo(batchRes.Orders[0], nil)

	exTPs := []domain.ExchangeOrder{}
	for _, tp := range batchRes.Orders[1 : 1+len(takeProfitSvcs)] {
		exOrder := newEo(tp, nil)
		exTPs = append(exTPs, exOrder)
	}

	exSLs := []domain.ExchangeOrder{}
	for _, sl := range batchRes.Orders[1+len(takeProfitSvcs):] {
		exOrder := newEo(sl, nil)
		exSLs = append(exSLs, exOrder)
	}

	return outbound.ExchangeOrders{
		Order:       newEo(batchRes.Orders[0], nil),
		TakeProfits: exTPs,
		StopLosses:  exSLs,
	}, errors.Join(errs...)
}

func (f *Exchange) ModifyOrders(ctx context.Context, req outbound.ModifyOrdersRequest) (outbound.ExchangeOrders, error) {
	orderMod := toOrderModification(req.Order)

	var tps, sls []futures.OrderModification
	for _, tpMod := range req.TakeProfit {
		mod := toOrderModification(tpMod)
		tps = append(tps, mod)
	}
	for _, slMod := range req.TakeProfit {
		mod := toOrderModification(slMod)
		sls = append(sls, mod)
	}

	batchOrders := []futures.OrderModification{orderMod}
	batchOrders = append(batchOrders, tps...)
	batchOrders = append(batchOrders, sls...)

	svc := f.client.NewModifyMultipleOrdersService().BatchOrders(batchOrders)

	resp, err := svc.Do(ctx)
	if err != nil {
		return outbound.ExchangeOrders{}, nil
	}

	orderResp, err := modRespAt(resp, 0)
	if err != nil {
		return outbound.ExchangeOrders{}, nil
	}

	fmt.Print(orderResp)
	return outbound.ExchangeOrders{
		Order:       newEo(orderResp, nil),
		TakeProfits: []domain.ExchangeOrder{}, //todo
		StopLosses:  []domain.ExchangeOrder{},
	}, nil
}

func (f *Exchange) CancelOrders(ctx context.Context, req outbound.CancelOrdersRequest) error {
	o := req.ExchangeOrder.(*exchangeOrder)
	orderIDs := []int64{o.order.OrderID}
	for _, eorder := range req.TPSLExchangeOrders {
		o := eorder.(*exchangeOrder)
		orderIDs = append(orderIDs, o.order.OrderID)
	}

	f.logger.Debugf("attempting to cancel %d orders", len(orderIDs))

	// assume all have the same symbol
	_, err := f.client.NewCancelMultipleOrdersService().Symbol(o.order.Symbol).OrderIDList(orderIDs).Do(ctx)
	return err
}

func (f *Exchange) cancelOrder(ctx context.Context, symbol string, id int64) error {
	_, err := f.client.NewCancelOrderService().Symbol(symbol).OrderID(id).Do(ctx)
	return err
}
