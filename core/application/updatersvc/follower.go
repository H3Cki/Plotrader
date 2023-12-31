package updatersvc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"go.uber.org/zap"
)

type orderTicker struct {
	order  *domain.Order
	ticker *time.Ticker
}

type updater struct {
	logger    *zap.SugaredLogger
	orders    []orderTicker
	publisher outbound.Publisher

	mu sync.Mutex
}

func newFollower(logger *zap.SugaredLogger, publisher outbound.Publisher) *updater {
	return &updater{
		logger:    logger,
		orders:    []orderTicker{},
		publisher: publisher,
	}
}

func (f *updater) createOrder(ctx context.Context, order *domain.Order, exchange outbound.Exchange) error {
	eo, err := f.createExchangeOrder(order, exchange)
	if err != nil {
		return err
	}

	order.ExchangeOrders = append(order.ExchangeOrders, eo)

	if err := f.publisher.PublishOrderUpdate(ctx, outbound.OrderUpdate{
		Order: *order,
	}); err != nil {
		f.logger.Error(err)
	}

	fo := orderTicker{
		order:  order,
		ticker: time.NewTicker(order.Params.Interval),
	}

	f.addOrderTicker(fo)

	go func() {
		tick := <-time.After(time.Until(NextStartTime(time.Now(), order.Params.Interval)))
		for {
			latest := order.ExchangeOrders[len(order.ExchangeOrders)-1]
			eo, err = f.updatePrice(ctx, tick, order, exchange, latest)
			if err != nil {
				fo.ticker.Stop()
				return
			}

			order.ExchangeOrders = append(order.ExchangeOrders, eo)

			if err := f.publisher.PublishOrderUpdate(ctx, outbound.OrderUpdate{
				Order: *order,
			}); err != nil {
				f.logger.Error(err)
			}

			var ok bool
			tick, ok = <-fo.ticker.C
			if !ok {
				break
			}
		}

	}()

	return nil
}

func (f *updater) updatePrice(ctx context.Context, t time.Time, req *domain.Order, exchange outbound.Exchange, eo domain.ExchangeOrders) (next domain.ExchangeOrders, err error) {
	order, takeProfit, stopLoss, err := orderDetails(req, t)
	if err != nil {
		return domain.ExchangeOrders{}, err
	}

	if err := exchange.CancelOrder(ctx, outbound.CancelOrderRequest{
		Pair:         req.Params.Pair,
		OrderID:      eo.Order.ID,
		TakeProfitID: eo.TakeProfit.ID,
		StopLossID:   eo.StopLoss.ID,
	}); err != nil {
		return domain.ExchangeOrders{}, err
	}

	return exchange.CreateOrder(context.Background(), outbound.CreateOrderRequest{
		Pair:       req.Params.Pair,
		Side:       req.Params.Side,
		Order:      order,
		TakeProfit: takeProfit,
		StopLoss:   stopLoss,
	})
}

func (f *updater) cancelOrder(orderID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, fo := range f.orders {
		if fo.order.ID == orderID {
			fo.ticker.Stop()
			f.orders = slices.Delete(f.orders, i, i+1)
			return nil
		}
	}

	return domain.ErrOrderNotFound
}

func (f *updater) createExchangeOrder(req *domain.Order, exchange outbound.Exchange) (domain.ExchangeOrders, error) {
	symbolPrice := 0.0
	if !req.Params.DisableProtection {
		sp, err := exchange.GetPrice(context.Background(), outbound.GetPriceRequest{Pair: req.Params.Pair})
		if err != nil {
			return domain.ExchangeOrders{}, fmt.Errorf("error getting symbol price: %v", err)
		}

		symbolPrice = sp
	}

	t := time.Now()

	order, takeProfit, stopLoss, err := orderDetails(req, t)
	if err != nil {
		return domain.ExchangeOrders{}, err
	}

	if !req.Params.DisableProtection {
		switch req.Params.Side {
		case domain.OrderSideBuy:
			if symbolPrice < order.Price {
				return domain.ExchangeOrders{}, errors.New("price protection: price exceeded")
			}
		case domain.OrderSideSell:
			if symbolPrice > order.Price {
				return domain.ExchangeOrders{}, errors.New("price protection: price exceeded")
			}
		}
	}

	return exchange.CreateOrder(context.Background(), outbound.CreateOrderRequest{
		Pair:       req.Params.Pair,
		Side:       req.Params.Side,
		Order:      order,
		TakeProfit: takeProfit,
		StopLoss:   stopLoss,
	})
}

func (u *updater) addOrderTicker(ot orderTicker) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.orders = append(u.orders, ot)
}

func orderDetails(req *domain.Order, t time.Time) (orderDetails outbound.OrderDetails, takeProfit *outbound.StopDetails, stopLoss *outbound.StopDetails, err error) {
	errs := []error{}

	orderPrice, err := req.Params.Order.Plot.At(t)
	if err != nil {
		errs = append(errs, err)
	}

	orderDetails = outbound.OrderDetails{
		Type:         req.Params.Order.Type,
		TimeInForce:  req.Params.Order.TimeInForce,
		BaseQuantity: baseQuantity(orderPrice, req.Params.Order.BaseQuantity, req.Params.Order.QuoteQuantity),
		Price:        orderPrice,
	}

	var tp, sl *outbound.StopDetails

	if req.Params.TakeProfit != nil {
		tpPrice, err := req.Params.TakeProfit.Plot.At(t)
		if err != nil {
			errs = append(errs, err)
		} else {
			tp = &outbound.StopDetails{
				Type:        req.Params.TakeProfit.Type,
				TimeInForce: req.Params.TakeProfit.TimeInForce,
				QuantityPct: req.Params.TakeProfit.QuantityPct,
				Price:       tpPrice,
			}
		}
	}

	if req.Params.StopLoss != nil {
		slPrice, err := req.Params.StopLoss.Plot.At(t)
		if err != nil {
			errs = append(errs, err)
		} else {
			sl = &outbound.StopDetails{
				Type:        req.Params.StopLoss.Type,
				TimeInForce: req.Params.StopLoss.TimeInForce,
				QuantityPct: req.Params.StopLoss.QuantityPct,
				Price:       slPrice,
			}
		}
	}

	if len(errs) > 0 {
		return outbound.OrderDetails{}, nil, nil, errors.Join(errs...)
	}

	return orderDetails, tp, sl, nil
}

func baseQuantity(price, base, quote float64) float64 {
	if base != 0 {
		return base
	}

	return quote / price
}
