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

func (f *Exchange) GetOrder(ctx context.Context, deo domain.ExchangeOrder) (domain.ExchangeOrder, error) {
	eo, ok := deo.(exchangeOrder)
	if !ok {
		return nil, errors.New("unexpected ExchangeOrder type")
	}

	order, err := f.client.NewGetOrderService().OrderID(eo.O.OrderID).Symbol(eo.O.Symbol).Do(ctx)
	if err != nil {
		return nil, err
	}
	return newEo(*order), nil
}

// Order
func (f *Exchange) CreateOrder(ctx context.Context, req outbound.CreateOrderRequest) (domain.ExchangeOrder, error) {
	resp, err := f.createOrder(ctx, req)
	return resp, convertErr(err)
}

func (f *Exchange) ModifyOrder(ctx context.Context, req outbound.ModifyOrderRequest) (domain.ExchangeOrder, error) {
	resp, err := f.modifyOrder(ctx, req)
	return resp, convertErr(err)
}

// TakeProfit
func (f *Exchange) CreateTakeProfitOrder(ctx context.Context, req outbound.CreateTakeProfitRequest) (domain.ExchangeOrder, error) {
	resp, err := f.createTP(ctx, req)
	return resp, convertErr(err)
}

func (f *Exchange) ModifyTakeProfitOrder(ctx context.Context, req outbound.ModifyTakeProfitRequest) (domain.ExchangeOrder, error) {
	resp, err := f.modifyTP(ctx, req)
	return resp, convertErr(err)
}

// StopLoss
func (f *Exchange) CreateStopLossOrder(ctx context.Context, req outbound.CreateStopLossRequest) (domain.ExchangeOrder, error) {
	resp, err := f.createSL(ctx, req)
	return resp, convertErr(err)
}

func (f *Exchange) ModifyStopLossOrder(ctx context.Context, req outbound.ModifyStopLossRequest) (domain.ExchangeOrder, error) {
	resp, err := f.modifySL(ctx, req)
	return resp, convertErr(err)
}

func (f *Exchange) CancelOrder(ctx context.Context, eo domain.ExchangeOrder) error {
	err := f.cancelOrder(ctx, eo)
	return convertErr(err)
}

func (f *Exchange) createOrder(ctx context.Context, req outbound.CreateOrderRequest) (domain.ExchangeOrder, error) {
	symbol, err := f.symbol(ctx, pairToSymbol(req.Pair))
	if err != nil {
		return nil, err
	}

	side, err := orderSide(req.PosSide, false)
	if err != nil {
		return nil, err
	}

	ov := orderValues{
		symbol:       symbol,
		side:         side,
		orderType:    futures.OrderTypeLimit,
		price:        req.Request.Price,
		baseQuantity: req.Request.BaseQuantity,
		timeInForce:  futures.TimeInForceTypeGTC,
	}

	if err := applyFilters(&ov); err != nil {
		return nil, err
	}

	svc := f.client.NewCreateOrderService().Symbol(symbol.Symbol).
		Side(ov.side).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(fmt.Sprint(ov.baseQuantity)).
		Price(fmt.Sprint(ov.price))

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}

	return newEo(*corToOrder(resp)), nil
}

func (f *Exchange) createTP(ctx context.Context, req outbound.CreateTakeProfitRequest) (domain.ExchangeOrder, error) {
	parent, ok := req.Parent.(exchangeOrder)
	if !ok {
		return nil, errors.New("unexpected ExchangeOrder type")
	}

	symbol, err := f.symbol(ctx, parent.O.Symbol)
	if err != nil {
		return nil, err
	}

	ov := orderValues{
		symbol:       symbol,
		side:         oppositeSide(parent.O.Side),
		orderType:    futures.OrderTypeLimit,
		price:        req.Request.StopPrice,
		baseQuantity: req.Request.BaseQuantity,
		timeInForce:  futures.TimeInForceTypeGTC,
	}

	if err := applyFilters(&ov); err != nil {
		return nil, err
	}

	svc := f.client.NewCreateOrderService().Symbol(symbol.Symbol).
		Side(ov.side).
		Type(futures.OrderTypeTakeProfitMarket).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(fmt.Sprint(ov.baseQuantity)).
		StopPrice(fmt.Sprint(ov.price)).
		PriceProtect(false)

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}

	return newEo(*corToOrder(resp)), nil
}

func (f *Exchange) modifyTP(ctx context.Context, req outbound.ModifyTakeProfitRequest) (domain.ExchangeOrder, error) {
	eo, ok := req.ExchangeOrder.(exchangeOrder)
	if !ok {
		return nil, errors.New("unexpected ExchangeOrder type")
	}

	if eoCurrPrice, err := strconv.ParseFloat(eo.O.StopPrice, 64); err != nil && eoCurrPrice == req.Request.StopPrice {
		f.logger.Debugf("ignoring modification of TP %d, prev=%f, new=%f", eo.O.OrderID, eoCurrPrice, req.Request.StopPrice)
		return eo, nil
	}

	if err := f.cancelOrder(ctx, eo); err != nil {
		return nil, err
	}

	return f.createTP(ctx, outbound.CreateTakeProfitRequest{
		Parent: req.Parent,
		Request: outbound.TakeProfitRequest{
			BaseQuantity: req.Request.BaseQuantity,
			Price:        req.Request.Price,
			StopPrice:    req.Request.StopPrice,
		},
	})
}

func (f *Exchange) createSL(ctx context.Context, req outbound.CreateStopLossRequest) (domain.ExchangeOrder, error) {
	parent, ok := req.Parent.(exchangeOrder)
	if !ok {
		return nil, errors.New("unexpected ExchangeOrder type")
	}

	symbol, err := f.symbol(ctx, parent.O.Symbol)
	if err != nil {
		return nil, err
	}

	ov := orderValues{
		symbol:       symbol,
		side:         oppositeSide(parent.O.Side),
		orderType:    futures.OrderTypeLimit,
		price:        req.Request.StopPrice,
		baseQuantity: req.Request.BaseQuantity,
		timeInForce:  futures.TimeInForceTypeGTC,
	}

	if err := applyFilters(&ov); err != nil {
		return nil, err
	}

	svc := f.client.NewCreateOrderService().Symbol(symbol.Symbol).
		Side(ov.side).
		Type(futures.OrderTypeStopMarket).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(fmt.Sprint(ov.baseQuantity)).
		StopPrice(fmt.Sprint(ov.price)).
		PriceProtect(false)

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}

	return newEo(*corToOrder(resp)), nil
}

func (f *Exchange) modifySL(ctx context.Context, req outbound.ModifyStopLossRequest) (domain.ExchangeOrder, error) {
	eo, ok := req.ExchangeOrder.(exchangeOrder)
	if !ok {
		return nil, errors.New("unexpected ExchangeOrder type")
	}

	if eoCurrPrice, err := strconv.ParseFloat(eo.O.StopPrice, 64); err != nil && eoCurrPrice == req.Request.StopPrice {
		f.logger.Debugf("ignoring modification of SL %d, prev=%f, new=%f", eo.O.OrderID, eoCurrPrice, req.Request.StopPrice)
		return eo, nil
	}

	if err := f.cancelOrder(ctx, eo); err != nil {
		return nil, err
	}

	return f.createSL(ctx, outbound.CreateStopLossRequest{
		Parent: req.Parent,
		Request: outbound.StopLossRequest{
			BaseQuantity: req.Request.BaseQuantity,
			Price:        req.Request.Price,
			StopPrice:    req.Request.StopPrice,
		},
	})
}

func (f *Exchange) modifyOrder(ctx context.Context, req outbound.ModifyOrderRequest) (domain.ExchangeOrder, error) {
	eo, ok := req.ExchangeOrder.(exchangeOrder)
	if !ok {
		return nil, errors.New("unexpected ExchangeOrder type")
	}

	symbol, err := f.symbol(ctx, eo.O.Symbol)
	if err != nil {
		return nil, err
	}

	ov := orderValues{
		symbol:       symbol,
		side:         eo.O.Side,
		orderType:    futures.OrderTypeLimit,
		price:        req.Request.Price,
		baseQuantity: req.Request.BaseQuantity,
		timeInForce:  futures.TimeInForceTypeGTC,
	}

	if err := applyFilters(&ov); err != nil {
		return nil, err
	}

	svc := f.client.NewModifyOrderService().
		OrderID(eo.O.OrderID).
		Price(ov.price).
		Quantity(ov.baseQuantity).
		Side(ov.side).
		Symbol(ov.symbol.Symbol)

	resp, err := svc.Do(ctx)
	// If error is "no need to modify the order, ignore err"
	apiErr, ok := err.(*common.APIError)
	if ok && apiErr.Code == -5027 {
		f.logger.Debugf("ignoring modification of order %d, prev=%s, new=%f", eo.O.OrderID, eo.O.Price, ov.price)
		return eo, nil
	}
	if err != nil {
		return nil, err
	}

	return newEo(*morToOrder(resp)), nil
}

type orderValues struct {
	symbol       futures.Symbol
	side         futures.SideType
	orderType    futures.OrderType
	price        float64
	baseQuantity float64
	timeInForce  futures.TimeInForceType
}

func (f *Exchange) cancelOrder(ctx context.Context, eo domain.ExchangeOrder) error {
	e, ok := eo.(exchangeOrder)
	if !ok {
		return fmt.Errorf("unexpected order type")
	}

	if e.O.Type != futures.OrderTypeLimit {
		svc := f.client.NewCancelOrderService()
		_, err := svc.OrderID(e.O.OrderID).Symbol(e.O.Symbol).Do(ctx)
		return err
	}

	if e.O.Status == futures.OrderStatusTypeFilled {
		_, err := f.client.NewGetPositionRiskService().Symbol(e.O.Symbol).Do(ctx)
		if err != nil {
			return err
		}

		svc := f.client.NewCreateOrderService().
			Quantity(e.O.ExecutedQuantity).
			PositionSide(e.O.PositionSide).
			Side(oppositeSide(e.O.Side)).
			Symbol(e.O.Symbol).
			Price(e.O.Price).
			Type(e.O.Type).TimeInForce(e.O.TimeInForce)
		_, err = svc.Do(ctx)
		return err
	}

	res, err := f.client.NewGetOrderService().Symbol(e.O.Symbol).OrderID(e.O.OrderID).Do(ctx)
	if err != nil {
		return err
	}

	if res.Status == futures.OrderStatusTypeFilled {
		svc := f.client.NewCreateOrderService().
			Quantity(res.ExecutedQuantity).
			PositionSide(res.PositionSide).
			Side(oppositeSide(res.Side)).
			Symbol(res.Symbol).
			Price(res.Price).
			Type(res.Type).TimeInForce(res.TimeInForce)
		_, err = svc.Do(ctx)
		return err
	}

	svc := f.client.NewCancelOrderService()
	_, err = svc.OrderID(e.O.OrderID).Symbol(e.O.Symbol).Do(ctx)
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

func convertErr(err error) error {
	if err == nil {
		return nil
	}
	if api, ok := err.(*common.APIError); ok {
		switch api.Code {
		case -4016:
			return outbound.ErrPriceOutOfRange
		}
	}
	return err
}
