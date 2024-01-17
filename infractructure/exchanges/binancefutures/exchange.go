package binancefutures

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/H3Cki/go-binance/v2/common"
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
	ExchangeInfoer outbound.FileLoader[ExchangeInfo]
	UserConfig     UserConfig
}

type Exchange struct {
	logger *zap.SugaredLogger
	client *futures.Client
	ei     ExchangeInfo
	eier   outbound.FileLoader[ExchangeInfo]
}

type ExchangeInfo futures.ExchangeInfo

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

func (f *Exchange) GetOrder(ctx context.Context, req outbound.GetExchangeOrderRequest) (*domain.ExchangeOrder, error) {
	eo := req.EO
	order, err := f.client.NewGetOrderService().OrderID(eo.ID.(int64)).Symbol(eo.Symbol).Do(ctx)
	if err != nil {
		return nil, err
	}
	return orderToOrder(order)
}

// Order
func (e *Exchange) CreateOrder(ctx context.Context, req outbound.CreateExchangeOrderRequest) (*domain.ExchangeOrder, error) {
	symbol, err := e.symbol(ctx, pairToSymbol(req.Pair))
	if err != nil {
		return nil, err
	}

	side, err := orderSide(req.Side)
	if err != nil {
		return nil, err
	}

	orderType, err := orderType(req.Type)
	if err != nil {
		return nil, err
	}

	ov := orderValues{
		symbol:       symbol,
		side:         side,
		orderType:    orderType,
		price:        req.Price,
		stopPrice:    req.StopPrice,
		baseQuantity: req.BaseQuantity,
		timeInForce:  futures.TimeInForceTypeGTC,
	}

	resp, err := e.createOrder(ctx, ov)
	return resp, err
}

func (f *Exchange) ModifyOrder(ctx context.Context, req outbound.ModifyExchangeOrderRequest) (*domain.ExchangeOrder, error) {
	resp, err := f.modifyOrder(ctx, req)
	return resp, err
}

func (f *Exchange) CancelOrder(ctx context.Context, req outbound.CancelExchangeOrdersRequest) (*domain.ExchangeOrder, error) {
	return f.cancelOrder(ctx, req)
}

func (e *Exchange) createOrder(ctx context.Context, ov orderValues) (*domain.ExchangeOrder, error) {
	if err := applyFilters(&ov); err != nil {
		return nil, fmt.Errorf("filter error: %w", err)
	}

	switch ov.orderType {
	case futures.OrderTypeLimit:
		return e.createLimit(ctx, ov)
	case futures.OrderTypeStopMarket:
		return e.createStopMarket(ctx, ov)
	case futures.OrderTypeTakeProfitMarket:
		return e.createTakeProfitMarket(ctx, ov)
	}

	return nil, fmt.Errorf("unsupported order type %s", ov.orderType)
}

func (e *Exchange) createLimit(ctx context.Context, ov orderValues) (*domain.ExchangeOrder, error) {
	svc := e.client.NewCreateOrderService().
		Symbol(ov.symbol.Symbol).
		Side(ov.side).
		Type(ov.orderType).
		TimeInForce(ov.timeInForce).
		Quantity(fmt.Sprint(ov.baseQuantity)).
		Price(fmt.Sprint(ov.price))

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}

	return createRespToOrder(resp)
}

func (e *Exchange) createStopMarket(ctx context.Context, ov orderValues) (*domain.ExchangeOrder, error) {
	svc := e.client.NewCreateOrderService().
		Symbol(ov.symbol.Symbol).
		Side(ov.side).
		Type(ov.orderType).
		TimeInForce(ov.timeInForce).
		Quantity(fmt.Sprint(ov.baseQuantity)).
		StopPrice(fmt.Sprint(ov.price))

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}

	return createRespToOrder(resp)
}

func (e *Exchange) createTakeProfitMarket(ctx context.Context, ov orderValues) (*domain.ExchangeOrder, error) {
	svc := e.client.NewCreateOrderService().
		Symbol(ov.symbol.Symbol).
		Side(ov.side).
		Type(ov.orderType).
		TimeInForce(ov.timeInForce).
		Quantity(fmt.Sprint(ov.baseQuantity)).
		StopPrice(fmt.Sprint(ov.price))

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}

	return createRespToOrder(resp)
}

func (f *Exchange) modifyOrder(ctx context.Context, req outbound.ModifyExchangeOrderRequest) (*domain.ExchangeOrder, error) {
	eo := req.EO

	symbol, err := f.symbol(ctx, eo.Symbol)
	if err != nil {
		return nil, err
	}

	ov := orderValues{
		symbol:       symbol,
		side:         futures.SideType(eo.Side),
		orderType:    futures.OrderTypeLimit,
		price:        req.Price,
		baseQuantity: req.BaseQuantity,
		timeInForce:  futures.TimeInForceTypeGTC,
	}

	if err := applyFilters(&ov); err != nil {
		return nil, err
	}

	if ov.price == eo.Price && ov.baseQuantity == eo.BaseQuantity {
		f.logger.Debugf("ignoring modification of order %d, prev=%s, new=%f", eo.ID, eo.Price, ov.price)
		return eo, nil
	}

	switch ov.orderType {
	case futures.OrderTypeLimit:
		svc := f.client.NewModifyOrderService().
			OrderID(eo.ID.(int64)).
			Price(ov.price).
			Quantity(ov.baseQuantity).
			Side(ov.side).
			Symbol(ov.symbol.Symbol)

		resp, err := svc.Do(ctx)
		// If error is "no need to modify the order, ignore err"
		apiErr, ok := err.(*common.APIError)
		if ok && apiErr.Code == -5027 {
			f.logger.Debugf("ignoring modification of order %d, prev=%s, new=%f", eo.ID, eo.Price, ov.price)
			return eo, nil
		}
		if err != nil {
			return nil, err
		}

		return modifyRespToOrder(resp)
	}

	return f.recreateOrder(ctx, eo, ov)
}

func (f *Exchange) recreateOrder(ctx context.Context, eo *domain.ExchangeOrder, ov orderValues) (*domain.ExchangeOrder, error) {
	_, err := f.cancelOrder(ctx, outbound.CancelExchangeOrdersRequest{EO: eo})
	if err != nil {
		return nil, err
	}
	return f.createOrder(ctx, ov)
}

func (f *Exchange) cancelOrder(ctx context.Context, req outbound.CancelExchangeOrdersRequest) (*domain.ExchangeOrder, error) {
	eo := req.EO
	svc := f.client.NewCancelOrderService()
	resp, err := svc.OrderID(eo.ID.(int64)).Symbol(eo.Symbol).Do(ctx)
	if err != nil {
		return nil, err
	}
	return cancelRespToOrder(resp)
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
	_, err = f.info(ctx, true)
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

type orderValues struct {
	symbol       futures.Symbol
	side         futures.SideType
	orderType    futures.OrderType
	price        float64
	stopPrice    float64
	baseQuantity float64
	timeInForce  futures.TimeInForceType
}
