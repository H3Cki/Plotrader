package updatersvc

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"go.uber.org/zap"
)

type orderTicker struct {
	order  *domain.Follow
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

func (f *updater) createOrder(ctx context.Context, follow *domain.Follow, exchange outbound.Exchange) error {
	eos, err := f.createExchangeOrders(follow, exchange)
	if err != nil {
		return err
	}

	if err := applyEos(follow.Orders, eos); err != nil {
		return err
	}

	if err := f.publisher.PublishOrderUpdate(ctx, outbound.OrderUpdate{
		Follow: *follow,
	}); err != nil {
		f.logger.Error(err)
	}

	go func() {
		nextStart := NextStartTime(time.Now(), follow.Interval).Add(-500 * time.Millisecond) // some margin for standard delays
		f.logger.Debugf("next interval: %s", nextStart.String())

		tick := <-time.After(time.Until(nextStart))
		ticker := time.NewTicker(follow.Interval)

		fo := orderTicker{
			order:  follow,
			ticker: ticker,
		}

		f.addOrderTicker(fo)

		for {
			f.logger.Debugf("updating order %v", fo.order)
			eos, err = f.updatePrice(ctx, tick, follow.Orders, exchange)
			if err != nil {
				fo.ticker.Stop()
				f.logger.Errorf("error updating order price %v", err)
				return
			}

			if err := f.publisher.PublishOrderUpdate(ctx, outbound.OrderUpdate{
				Follow: *follow,
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

func applyEos(orders domain.Orders, eos outbound.ExchangeOrders) error {
	if len(orders.TakeProfits) != len(eos.TakeProfits) {
		return fmt.Errorf("take profits length mismatch: expected %d, got %d", len(orders.TakeProfits), len(eos.TakeProfits))
	}
	if len(orders.StopLosses) != len(eos.StopLosses) {
		return fmt.Errorf("stop losses length mismatch: expected %d, got %d", len(orders.TakeProfits), len(eos.TakeProfits))
	}
	orders.Order.ExchangeOrder = eos.Order
	for i := range orders.TakeProfits {
		orders.TakeProfits[i].ExchangeOrder = eos.TakeProfits[i]
	}
	for i := range orders.StopLosses {
		orders.StopLosses[i].ExchangeOrder = eos.StopLosses[i]
	}
	return nil
}

func (f *updater) updatePrice(ctx context.Context, t time.Time, orders domain.Orders, exchange outbound.Exchange) (outbound.ExchangeOrders, error) {
	orderPrice, err := orders.Order.Plot.At(t)
	if err != nil {
		return outbound.ExchangeOrders{}, err
	}
	orderMod := outbound.OrderModification{
		ExchangeOrder: orders.Order.ExchangeOrder,
		OrderDetails: outbound.OrderDetails{
			BaseQuantity: baseQuantity(orderPrice, orders.Order.BaseQuantity, orders.Order.QuoteQuantity),
			Price:        orderPrice,
		},
	}

	tpMods := []outbound.OrderModification{}
	for _, tp := range orders.TakeProfits {
		price, err := tp.Plot.At(t)
		if err != nil {
			return outbound.ExchangeOrders{}, err
		}
		tpMods = append(tpMods, outbound.OrderModification{
			ExchangeOrder: tp.ExchangeOrder,
			OrderDetails: outbound.OrderDetails{
				BaseQuantity: tp.ExchangeOrder.BaseQuantity(),
				StopPrice:    price,
			},
		})
	}

	slMods := []outbound.OrderModification{}
	for _, sl := range orders.StopLosses {
		price, err := sl.Plot.At(t)
		if err != nil {
			return outbound.ExchangeOrders{}, err
		}
		slMods = append(slMods, outbound.OrderModification{
			ExchangeOrder: sl.ExchangeOrder,
			OrderDetails: outbound.OrderDetails{
				BaseQuantity: sl.ExchangeOrder.BaseQuantity(),
				StopPrice:    price,
			},
		})
	}

	modReq := outbound.ModifyOrdersRequest{
		Order:      orderMod,
		TakeProfit: tpMods,
		StopLoss:   slMods,
	}

	return exchange.ModifyOrders(ctx, modReq)
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

func (f *updater) createExchangeOrders(req *domain.Follow, exchange outbound.Exchange) (outbound.ExchangeOrders, error) {
	t := time.Now()
	order, takeProfit, stopLoss, err := ordersAt(req.Orders, t)
	if err != nil {
		return outbound.ExchangeOrders{}, err
	}

	return exchange.CreateOrders(context.Background(), outbound.CreateOrdersRequest{
		Pair:       req.Pair,
		Side:       req.PositionSide,
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

func ordersAt(req domain.Orders, t time.Time) (order outbound.OrderDetails, takeProfits []outbound.OrderDetails, stopLosses []outbound.OrderDetails, err error) {
	orderPrice, err := req.Order.Plot.At(t)
	if err != nil {
		return outbound.OrderDetails{}, []outbound.OrderDetails{}, []outbound.OrderDetails{}, err
	}
	order = outbound.OrderDetails{
		BaseQuantity: baseQuantity(orderPrice, req.Order.BaseQuantity, req.Order.QuoteQuantity),
		Price:        orderPrice,
	}

	var tps, sls []outbound.OrderDetails

	for _, tp := range req.TakeProfits {
		price, err := tp.Plot.At(t)
		if err != nil {
			return outbound.OrderDetails{}, []outbound.OrderDetails{}, []outbound.OrderDetails{}, err
		}
		tps = append(tps, outbound.OrderDetails{
			BaseQuantity: order.BaseQuantity * tp.QuantityPct,
			StopPrice:    price,
		})
	}

	for _, sl := range req.StopLosses {
		price, err := sl.Plot.At(t)
		if err != nil {
			return outbound.OrderDetails{}, []outbound.OrderDetails{}, []outbound.OrderDetails{}, err
		}
		sls = append(sls, outbound.OrderDetails{
			BaseQuantity: order.BaseQuantity * sl.QuantityPct,
			StopPrice:    price,
		})
	}

	return order, tps, sls, nil
}

func baseQuantity(price, base, quote float64) float64 {
	if base != 0 {
		return base
	}

	return quote / price
}
