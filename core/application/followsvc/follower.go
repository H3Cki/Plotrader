package followsvc

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"go.uber.org/zap"
)

type worker struct {
	logger   *zap.SugaredLogger
	exchange outbound.Exchange
	follow   *domain.Follow
}

func (f *worker) createExchangeOrders(ctx context.Context, t time.Time) (*domain.Follow, error) {
	orders := f.follow.Orders

	createdOrder, err := f.createOrder(ctx, t)
	if err != nil {
		return nil, err
	}
	orders.Order = createdOrder

	for i, tp := range orders.TakeProfits {
		createdTP, err := f.createTP(ctx, t, tp, createdOrder.ExchangeOrder)
		if err != nil {
			return nil, err
		}
		orders.TakeProfits[i] = createdTP
	}

	for i, sl := range orders.StopLosses {
		createdSL, err := f.createSL(ctx, t, sl, createdOrder.ExchangeOrder)
		if err != nil {
			return nil, err
		}
		orders.StopLosses[i] = createdSL
	}

	f.follow.Orders = orders

	return f.follow, nil
}

func (w *worker) createOrder(ctx context.Context, t time.Time) (domain.Order, error) {
	order := w.follow.Orders.Order

	price, err := order.Plot.At(t)
	if err != nil {
		return domain.Order{}, err
	}

	eo, err := w.exchange.CreateOrder(ctx, outbound.CreateOrderRequest{
		Pair: w.follow.Pair,
		Side: w.follow.PositionSide,
		Request: outbound.OrderRequest{
			BaseQuantity: baseQuantity(price, order.BaseQuantity, order.QuoteQuantity),
			Price:        price,
		},
	})
	if err != nil {
		return domain.Order{}, err
	}
	order.ExchangeOrder = eo
	w.follow.Orders.Order = order
	return order, nil
}

func (w *worker) createTP(ctx context.Context, t time.Time, tp domain.TPSLOrder, parent domain.ExchangeOrder) (domain.TPSLOrder, error) {
	price, err := tp.Plot.At(t)
	if err != nil {
		return domain.TPSLOrder{}, err
	}

	eo, err := w.exchange.CreateTakeProfitOrder(ctx, outbound.CreateTakeProfitRequest{
		Parent: parent,
		Request: outbound.TakeProfitRequest{
			BaseQuantity: parent.BaseQuantity() * tp.QuantityPct,
			StopPrice:    price,
		},
	})
	if err != nil {
		return domain.TPSLOrder{}, err
	}

	tp.ExchangeOrder = eo
	return tp, nil
}

func (w *worker) createSL(ctx context.Context, t time.Time, sl domain.TPSLOrder, parent domain.ExchangeOrder) (domain.TPSLOrder, error) {
	price, err := sl.Plot.At(t)
	if err != nil {
		return domain.TPSLOrder{}, err
	}

	eo, err := w.exchange.CreateStopLossOrder(ctx, outbound.CreateStopLossRequest{
		Parent: parent,
		Request: outbound.StopLossRequest{
			BaseQuantity: parent.BaseQuantity() * sl.QuantityPct,
			StopPrice:    price,
		},
	})
	if err != nil {
		return domain.TPSLOrder{}, err
	}

	sl.ExchangeOrder = eo
	return sl, nil
}

func (f *worker) updateOrder(ctx context.Context, t time.Time, order domain.Order) (domain.Order, error) {
	if err := f.refreshOrder(ctx, &order); err != nil {
		f.logger.Errorf("error refreshing order %v: %v", order, err)
	}

	if order.ExchangeOrder.Status() != domain.ExchangeOrderStatusOpen {
		return order, nil
	}

	price, err := order.Plot.At(t)
	if err != nil {
		return domain.Order{}, err
	}

	eo, err := f.exchange.ModifyOrder(ctx, outbound.ModifyOrderRequest{
		ExchangeOrder: order.ExchangeOrder,
		Request: outbound.OrderRequest{
			BaseQuantity: order.ExchangeOrder.BaseQuantity(),
			Price:        price,
		},
	})
	if err != nil {
		return domain.Order{}, err
	}

	order.ExchangeOrder = eo

	return order, nil
}

func (f *worker) updateTP(ctx context.Context, t time.Time, tp domain.TPSLOrder, parent domain.ExchangeOrder) (domain.TPSLOrder, error) {
	price, err := tp.Plot.At(t)
	if err != nil {
		return domain.TPSLOrder{}, err
	}

	eo, err := f.exchange.ModifyTakeProfitOrder(ctx, outbound.ModifyTakeProfitRequest{
		Parent:        parent,
		ExchangeOrder: tp.ExchangeOrder,
		Request: outbound.TakeProfitRequest{
			BaseQuantity: tp.ExchangeOrder.BaseQuantity(),
			Price:        price,
			StopPrice:    price,
		},
	})

	tp.ExchangeOrder = eo
	return tp, nil
}

func (f *worker) updateSL(ctx context.Context, t time.Time, sl domain.TPSLOrder, parent domain.ExchangeOrder) (domain.TPSLOrder, error) {
	price, err := sl.Plot.At(t)
	if err != nil {
		return domain.TPSLOrder{}, err
	}

	eo, err := f.exchange.ModifyStopLossOrder(ctx, outbound.ModifyStopLossRequest{
		Parent:        parent,
		ExchangeOrder: sl.ExchangeOrder,
		Request: outbound.StopLossRequest{
			BaseQuantity: sl.ExchangeOrder.BaseQuantity(),
			Price:        price,
			StopPrice:    price,
		},
	})

	sl.ExchangeOrder = eo
	return sl, nil
}

func (f *worker) updateOrders(ctx context.Context, t time.Time) (*domain.Follow, error) {
	updatedOrder, err := f.updateOrder(ctx, t, f.follow.Orders.Order)
	if err != nil {
		return nil, err
	}

	f.follow.Orders.Order = updatedOrder

	for i, tp := range f.follow.Orders.TakeProfits {
		updatedTP, err := f.updateTP(ctx, t, tp, updatedOrder.ExchangeOrder)
		if err != nil {
			return nil, err
		}
		f.follow.Orders.TakeProfits[i] = updatedTP
	}

	for i, sl := range f.follow.Orders.StopLosses {
		updatedSL, err := f.updateSL(ctx, t, sl, updatedOrder.ExchangeOrder)
		if err != nil {
			return nil, err
		}
		f.follow.Orders.StopLosses[i] = updatedSL
	}

	return f.follow, nil
}

func (f *worker) cancelExchangeOrders(ctx context.Context) error {
	eosList := []domain.ExchangeOrder{f.follow.Orders.Order.ExchangeOrder}
	for _, tp := range f.follow.Orders.TakeProfits {
		eosList = append(eosList, tp.ExchangeOrder)
	}
	for _, sl := range f.follow.Orders.StopLosses {
		eosList = append(eosList, sl.ExchangeOrder)
	}
	return f.exchange.CancelOrders(ctx, outbound.CancelOrdersRequest{
		ExchangeOrders: eosList,
	})
}

func (f *worker) refreshOrder(ctx context.Context, order *domain.Order) error {
	freshEo, err := f.exchange.GetOrder(ctx, order.ExchangeOrder)
	if err != nil {
		return err
	}

	order.ExchangeOrder = freshEo
	return nil
}

type followTicker struct {
	follow *domain.Follow
	ticker *time.Ticker
}

type follower struct {
	logger        *zap.SugaredLogger
	followTickers []followTicker
	publisher     outbound.Publisher

	mu sync.Mutex
}

func newFollower(logger *zap.SugaredLogger, publisher outbound.Publisher) *follower {
	return &follower{
		logger:        logger,
		followTickers: []followTicker{},
		publisher:     publisher,
	}
}

func (f *follower) startFollow(ctx context.Context, follow *domain.Follow, exchange outbound.Exchange) error {
	w := worker{
		logger:   f.logger,
		exchange: exchange,
		follow:   follow,
	}

	followSnapshot, err := w.createExchangeOrders(ctx, time.Now())
	if err != nil {
		if err := w.cancelExchangeOrders(ctx); err != nil {
			f.logger.Errorf("error cancellign requests: %v", err)
		}
		return err
	}

	if err := f.publisher.PublishFollowUpdate(ctx, outbound.FollowUpdate{
		Follow: *followSnapshot,
	}); err != nil {
		f.logger.Error(err)
	}

	go func() {
		if err := f.run(ctx, w); err != nil {
			f.logger.Errorf("following error: %v", err)
		}
	}()

	return nil
}

func (f *follower) run(ctx context.Context, w worker) error {
	defer func() {
		f.logger.Infof("follow %s finished", w.follow.ID)
	}()

	nextStart := NextStartTime(time.Now(), w.follow.Interval).Add(-500 * time.Millisecond) // some margin for standard delays
	f.logger.Debugf("next interval: %s", nextStart.String())

	tick := <-time.After(time.Until(nextStart))
	ticker := time.NewTicker(w.follow.Interval)

	fo := followTicker{
		follow: w.follow,
		ticker: ticker,
	}

	f.addOrderTicker(fo)

	for {
		f.logger.Debugf("updating order %s", fo.follow.ID)
		followSnapshot, updateErr := w.updateOrders(ctx, tick)
		if updateErr != nil {
			f.logger.Errorf("error updating order: %v", updateErr)
			return updateErr
		}

		if err := f.publisher.PublishFollowUpdate(ctx, outbound.FollowUpdate{
			Follow: *followSnapshot,
		}); err != nil {
			f.logger.Error(err)
		}

		nextStart := NextStartTime(time.Now(), w.follow.Interval).Add(-500 * time.Millisecond) // some margin for standard delays
		f.logger.Debugf("next interval: %s", nextStart.String())

		var ok bool
		tick, ok = <-fo.ticker.C
		if !ok {
			break
		}
	}

	return nil
}

func (f *follower) stopFollow(followID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, fo := range f.followTickers {
		if fo.follow.ID == followID {
			fo.ticker.Stop()
			f.followTickers = slices.Delete(f.followTickers, i, i+1)
			return nil
		}
	}

	return domain.ErrOrderNotFound
}

func (u *follower) addOrderTicker(ot followTicker) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.followTickers = append(u.followTickers, ot)
}

func baseQuantity(price, base, quote float64) float64 {
	if base != 0 {
		return base
	}

	return quote / price
}
