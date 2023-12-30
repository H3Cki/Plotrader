package binancefutures

import (
	"context"
	"encoding/json"
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

var exchangeInfoFileName = "binancefutures_ei.json"

type Config struct {
	Testnet    bool   `json:"testnet"`
	API_KEY    string `json:"API_KEY" validate:"required"`
	SECRET_KEY string `json:"SECRET_KEY" validate:"required"`
}

type Exchange struct {
	logger *zap.SugaredLogger
	client *futures.Client
	ei     *futures.ExchangeInfo
}

func New(logger *zap.SugaredLogger, cfg Config) *Exchange {
	futures.UseTestnet = cfg.Testnet
	return &Exchange{
		logger: logger,
		client: futures.NewClient(cfg.API_KEY, cfg.SECRET_KEY),
	}
}

func (f *Exchange) Init(ctx context.Context) error {
	if err := f.client.NewPingService().Do(ctx); err != nil {
		return err
	}

	err := f.exchangeInfoFromFile()
	if errors.Is(err, os.ErrNotExist) || (err == nil && f.eiOutdated()) {
		f.logger.Info("loading exchange info from file")
		if err := f.exchangeInfo(ctx); err != nil {
			return fmt.Errorf("error loading exchange info: %w", err)
		}
	}

	return nil
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
	symbol, err := f.symbol(pairToSymbol(req.Pair))
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

func (f *Exchange) exchangeInfo(ctx context.Context) error {
	res, err := f.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return fmt.Errorf("unable to fetch spot exchange info: %w", err)
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("unable to marshal exchange info: %w", err)
	}

	err = os.WriteFile(exchangeInfoFileName, bytes, 0o777)
	if err != nil {
		f.logger.Errorf("unable to save exchange info to file: %w", err)
	}

	f.ei = res

	return nil
}

func (f *Exchange) exchangeInfoFromFile() error {
	bytes, err := os.ReadFile(exchangeInfoFileName)
	if err != nil {
		return err
	}

	ei := &futures.ExchangeInfo{}
	if err := json.Unmarshal(bytes, ei); err != nil {
		return err
	}

	f.ei = ei

	return nil
}

func (f *Exchange) eiOutdated() bool {
	return time.Since(time.Unix(f.ei.ServerTime, 0)) > time.Hour*24
}

func (f *Exchange) symbol(symbol string) (futures.Symbol, error) {
	fetched := false

	if f.eiOutdated() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := f.exchangeInfo(ctx); err != nil {
			f.logger.Errorf("error updating exchange info: %v", err)
		} else {
			fetched = true
		}
	}

	for _, fsymbol := range f.ei.Symbols {
		if fsymbol.Symbol == symbol {
			return fsymbol, nil
		}
	}

	// second chance, ei was loaded form file but the symbol might be new and require a reload
	if !fetched {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := f.exchangeInfo(ctx); err != nil {
			return futures.Symbol{}, err
		}

		for _, fsymbol := range f.ei.Symbols {
			if fsymbol.Symbol == symbol {
				return fsymbol, nil
			}
		}
	}

	return futures.Symbol{}, fmt.Errorf("unknown symbol: %s", symbol)
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
		baseQuantity: req.Order.BaseQuantity * req.TakeProfit.QuentityPct,
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
		baseQuantity: req.Order.BaseQuantity * req.StopLoss.QuentityPct,
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
