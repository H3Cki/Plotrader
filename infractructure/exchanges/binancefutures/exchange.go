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
	"github.com/adshao/go-binance/v2/futures"
	"go.uber.org/zap"
)

var (
	eiFileName = "binancefutures_ei.json"
	maxEiAge   = 24 * time.Hour
)

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

func (f *Exchange) CreateOrder(ctx context.Context, req outbound.CreateOrderRequest) (domain.ExchangeOrders, error) {
	symbol, err := f.symbol(ctx, pairToSymbol(req.Pair))
	if err != nil {
		return domain.ExchangeOrders{}, err
	}

	var takeProfitReq, stopLossReq *createOrderRequest

	errs := []error{}
	orderReq, err := toOrderRequest(req, symbol)
	if err != nil {
		errs = append(errs, err)
	}

	if req.TakeProfit != nil {
		takeProfitReq, err = toTakeProfitRequest(req, symbol)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if req.StopLoss != nil {
		stopLossReq, err = toStopLossRequest(req, symbol)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return domain.ExchangeOrders{}, errors.Join(errs...)
	}

	order, err := f.createOrder(ctx, orderReq)
	if err != nil {
		return domain.ExchangeOrders{}, err
	}

	var takeProfit *futures.CreateOrderResponse
	if takeProfitReq != nil {
		takeProfit, err = f.createOrder(ctx, takeProfitReq)
		if err != nil {
			if cancelErr := f.cancelOrder(ctx, cancelOrderRequest{
				id:     order.OrderID,
				symbol: order.Symbol,
			}); cancelErr != nil {
				return domain.ExchangeOrders{}, fmt.Errorf("%w: %s", err, cancelErr)
			}

			return domain.ExchangeOrders{}, err
		}
	}

	stopLoss, err := f.createOrder(ctx, stopLossReq)
	if err != nil {
		if cancelErr := f.cancelOrder(ctx, cancelOrderRequest{
			id:     order.OrderID,
			symbol: order.Symbol,
		}); cancelErr != nil {
			return domain.ExchangeOrders{}, fmt.Errorf("%w: %s", err, cancelErr)
		}

		if cancelTpErr := f.cancelOrder(ctx, cancelOrderRequest{
			id:     takeProfit.OrderID,
			symbol: takeProfit.Symbol,
		}); cancelTpErr != nil {
			return domain.ExchangeOrders{}, fmt.Errorf("%w: %s", err, cancelTpErr)
		}

		return domain.ExchangeOrders{}, err
	}

	exOrder, err := toExchangeOrder(order)
	if err != nil {
		f.logger.Errorf("error converting to ExchangeOrder: %w", err)
	}

	exTakeProfit, err := toExchangeOrder(takeProfit)
	if err != nil {
		f.logger.Errorf("error converting to ExchangeOrder: %w", err)
	}

	exStopLoss, err := toExchangeOrder(stopLoss)
	if err != nil {
		f.logger.Errorf("error converting to ExchangeOrder: %w", err)
	}

	return domain.ExchangeOrders{
		Order:      exOrder,
		TakeProfit: exTakeProfit,
		StopLoss:   exStopLoss,
	}, nil
}

func (f *Exchange) createOrder(ctx context.Context, req *createOrderRequest) (*futures.CreateOrderResponse, error) {
	if err := applyFilters(req); err != nil {
		return nil, err
	}

	createSvc := f.client.NewCreateOrderService().
		Symbol(req.symbol.Symbol).
		Side(req.side).
		Type(req.orderType).
		TimeInForce(req.timeInForce).
		Quantity(fmt.Sprint(req.baseQuantity)).
		Price(fmt.Sprint(req.price))

	return createSvc.Do(ctx)
}

func (f *Exchange) CancelOrder(ctx context.Context, req outbound.CancelOrderRequest) error {
	id, err := strconv.ParseInt(req.OrderID, 10, 64)
	if err != nil {
		return err
	}

	return f.cancelOrder(ctx, cancelOrderRequest{
		id:     id,
		symbol: pairToSymbol(req.Pair),
	})
}

type cancelOrderRequest struct {
	id     int64
	symbol string
}

func (f *Exchange) cancelOrder(ctx context.Context, req cancelOrderRequest) error {
	_, err := f.client.NewCancelOrderService().Symbol(req.symbol).OrderID(req.id).Do(ctx)
	return err
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

// info tries to read the ei from file, if it doesn't exist or is outdated it attempts to fetch the ei
func (f *Exchange) info(ctx context.Context, force bool) (updated bool, err error) {
	// Load if not exists
	ei, err := f.eier.Read(eiFileName)
	if force || os.IsNotExist(err) {
		ei, err = f.getExchangeInfo(ctx)
		if err != nil {
			return false, err
		}

		f.ei = ei

		// Ignore save error
		if err := f.eier.Save(eiFileName, ei); err != nil {
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
		if err := f.eier.Save(eiFileName, ei); err != nil {
			f.logger.Errorf("error saving exchange info: %v", err)
		}
	}

	return true, nil
}

func pairToSymbol(p domain.Pair) string {
	return p.Base + p.Quote
}

func parseSide(s domain.OrderSide) (futures.SideType, error) {
	switch s {
	case domain.OrderSideBuy:
		return futures.SideTypeBuy, nil
	case domain.OrderSideSell:
		return futures.SideTypeSell, nil
	}
	return "", fmt.Errorf("unexpected order side: %s", s)
}

func parseType(s domain.OrderType) (futures.OrderType, error) {
	switch s {
	case domain.OrderTypeLimit:
		return futures.OrderTypeLimit, nil
	case domain.OrderTypeTakeProfit:
		return futures.OrderTypeTakeProfitMarket, nil
	case domain.OrderTypeStopLoss:
		return futures.OrderTypeStopMarket, nil
	}
	return "", fmt.Errorf("unexpected order type: %s", s)
}

func parseTimeInForce(s domain.TimeInForce) (futures.TimeInForceType, error) {
	switch s {
	case domain.TimeInForceGTC:
		return futures.TimeInForceTypeGTC, nil
	}
	return "", fmt.Errorf("unexpected order time in force: %s", s)
}

func toOrderRequest(req outbound.CreateOrderRequest, symbol futures.Symbol) (*createOrderRequest, error) {
	orderSide, err := parseSide(req.Side)
	if err != nil {
		return nil, err
	}
	orderType, err := parseType(req.Order.Type)
	if err != nil {
		return nil, err
	}
	orderTIF, err := parseTimeInForce(req.Order.TimeInForce)
	if err != nil {
		return nil, err
	}

	return &createOrderRequest{
		symbol:       symbol,
		side:         orderSide,
		orderType:    orderType,
		price:        req.Order.Price,
		baseQuantity: req.Order.BaseQuantity,
		timeInForce:  orderTIF,
	}, nil
}

func toTakeProfitRequest(req outbound.CreateOrderRequest, symbol futures.Symbol) (*createOrderRequest, error) {
	orderSide, err := parseSide(req.Side) //???
	if err != nil {
		return nil, err
	}
	orderType, err := parseType(req.TakeProfit.Type)
	if err != nil {
		return nil, err
	}
	orderTIF, err := parseTimeInForce(req.TakeProfit.TimeInForce)
	if err != nil {
		return nil, err
	}

	return &createOrderRequest{
		symbol:       symbol,
		side:         orderSide,
		orderType:    orderType,
		price:        req.TakeProfit.Price,
		baseQuantity: req.Order.BaseQuantity * req.TakeProfit.QuantityPct,
		timeInForce:  orderTIF,
	}, nil
}

func toStopLossRequest(req outbound.CreateOrderRequest, symbol futures.Symbol) (*createOrderRequest, error) {
	orderSide, err := parseSide(req.Side) //???
	if err != nil {
		return nil, err
	}
	orderType, err := parseType(req.StopLoss.Type)
	if err != nil {
		return nil, err
	}
	orderTIF, err := parseTimeInForce(req.StopLoss.TimeInForce)
	if err != nil {
		return nil, err
	}

	return &createOrderRequest{
		symbol:       symbol,
		side:         orderSide,
		orderType:    orderType,
		price:        req.StopLoss.Price,
		baseQuantity: req.Order.BaseQuantity * req.StopLoss.QuantityPct,
		timeInForce:  orderTIF,
	}, nil
}

func toExchangeOrder(resp *futures.CreateOrderResponse) (domain.ExchangeOrder, error) {
	price, err := strconv.ParseFloat(resp.Price, 64)
	if err != nil {
		return domain.ExchangeOrder{}, err
	}

	baseQty, err := strconv.ParseFloat(resp.OrigQuantity, 64)
	if err != nil {
		return domain.ExchangeOrder{}, err
	}

	return domain.ExchangeOrder{
		ID:           fmt.Sprint(resp.OrderID),
		Price:        price,
		BaseQuantity: baseQty,
	}, nil
}
