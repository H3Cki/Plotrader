package followsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type follower struct {
	logger    *zap.SugaredLogger
	publisher outbound.Publisher
	loops     map[string]*intervalLoop
	registry  *registry
	mu        sync.Mutex
}

func newFollower(logger *zap.SugaredLogger, publisher outbound.Publisher) *follower {
	return &follower{
		logger:    logger,
		registry:  newRegistry(),
		publisher: publisher,
		loops:     map[string]*intervalLoop{},
	}
}

func (f *follower) newIntervalLoop(id string, itv, headstart time.Duration) *intervalLoop {
	loop := newIntervalLoop(f.logger, itv, headstart)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loops[id] = loop
	return loop
}

func (f *follower) startFollow(ctx context.Context, follow domain.Follow, exchange outbound.Exchange) (err error) {
	f.registry.setFollow(follow)

	f.logger.Debug("creating orders")

	intervalStart := IntervalStart(time.Now(), follow.Interval)
	if err := f.createOrders(ctx, follow.ID, intervalStart, exchange); err != nil {
		return err
	}

	f.publishFollowUpdate(ctx, follow.ID)

	loop := f.newIntervalLoop(follow.ID, follow.Interval, 1*time.Second)
	go func() {
		loop.loop(func(tick time.Time) error {
			err := f.handleTick(ctx, tick, follow.ID, exchange)
			if err != nil {
				f.logger.Errorf("error handling interval: %v", err)
			}
			if breakingError(err) {
				return err
			}
			return nil
		})
	}()

	return nil
}

func breakingError(err error) bool {
	switch {
	case errors.Is(err, outbound.ErrPriceOutOfRange):
		return false
	}
	return true
}

func (f *follower) handleTick(ctx context.Context, tick time.Time, followID string, ex outbound.Exchange) error {
	f.logger.Debugf("handling tick at %s", tick.String())

	uCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	defer func() {
		f.publishFollowUpdate(ctx, followID)
	}()

	err := f.updateOrders(uCtx, tick, followID, ex)

	if err != nil {
		f.logger.Debug("cancelling all orders due to error")
		if cErr := f.cancelOrders(ctx, followID, ex); cErr != nil {
			err = fmt.Errorf("%w : %w", err, cErr)
		}
	}

	return err
}

func (f *follower) cancelOrders(ctx context.Context, followID string, ex outbound.Exchange) error {
	follow := f.registry.getFollow(followID)
	eos := []domain.ExchangeOrder{follow.Order.ExchangeOrder}
	for _, tp := range follow.TakeProfits {
		eos = append(eos, tp.ExchangeOrder)
	}
	for _, sl := range follow.StopLosses {
		eos = append(eos, sl.ExchangeOrder)
	}
	errs := []error{}
	for _, eo := range eos {
		errs = append(errs, ex.CancelOrder(ctx, eo))
	}
	return errors.Join(errs...)
}

func (f *follower) createOrders(ctx context.Context, followID string, itvStart time.Time, ex outbound.Exchange) error {
	if err := f.createParentOrder(ctx, itvStart, followID, ex); err != nil {
		return err
	}
	return f.createStops(ctx, itvStart, followID, ex)
}

func (f *follower) createStops(ctx context.Context, t time.Time, followID string, ex outbound.Exchange) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		return f.createSLs(ctx, t, followID, ex)
	})
	eg.Go(func() error {
		return f.createTPs(ctx, t, followID, ex)
	})
	return eg.Wait()
}

func (f *follower) createTPs(ctx context.Context, t time.Time, followID string, ex outbound.Exchange) error {
	modder := f.registry.getModder(followID)
	follow := modder.getFollow()
	order := follow.Order

	eg := errgroup.Group{}
	for _, tp := range follow.TakeProfits {
		func(takeProfit domain.StopOrder) {
			eg.Go(func() error {
				price, err := takeProfit.Plot.At(t)
				if err != nil {
					return err
				}

				eo, err := ex.CreateTakeProfitOrder(ctx, outbound.CreateTakeProfitRequest{
					Parent: order.ExchangeOrder,
					Request: outbound.TakeProfitRequest{
						BaseQuantity: order.ExchangeOrder.BaseQuantity() * tp.QuantityPct,
						Price:        0,
						StopPrice:    price,
					},
				})
				if err != nil {
					return err
				}

				updatedStop := updateStopEo(takeProfit, eo)
				modder.setStop(updatedStop)
				return nil
			})
		}(tp)
	}
	return eg.Wait()
}

func updateStopEo(s domain.StopOrder, eo domain.ExchangeOrder) domain.StopOrder {
	s.ExchangeOrder = eo
	s.Status = domain.EoStatusToStopStatus(eo.Status())
	return s
}

func (f *follower) createSLs(ctx context.Context, t time.Time, followID string, ex outbound.Exchange) error {
	modder := f.registry.getModder(followID)
	follow := modder.getFollow()
	order := follow.Order

	eg := errgroup.Group{}
	for _, sl := range follow.StopLosses {
		func(stopLoss domain.StopOrder) {
			eg.Go(func() error {
				price, err := stopLoss.Plot.At(t)
				if err != nil {
					return err
				}

				eo, err := ex.CreateStopLossOrder(ctx, outbound.CreateStopLossRequest{
					Parent: order.ExchangeOrder,
					Request: outbound.StopLossRequest{
						BaseQuantity: order.ExchangeOrder.BaseQuantity() * sl.QuantityPct,
						Price:        0,
						StopPrice:    price,
					},
				})
				if err != nil {
					return err
				}

				updatedStop := updateStopEo(stopLoss, eo)
				modder.setStop(updatedStop)
				return nil
			})
		}(sl)
	}
	return eg.Wait()
}

func (f *follower) createParentOrder(ctx context.Context, t time.Time, followID string, ex outbound.Exchange) error {
	modder := f.registry.getModder(followID)
	follow := modder.getFollow()

	price, err := follow.Order.Plot.At(t)
	if err != nil {
		return err
	}

	f.logger.Debug("creating order on exchange")
	eo, err := ex.CreateOrder(ctx, outbound.CreateOrderRequest{
		Pair:    follow.Pair,
		PosSide: follow.Side,
		Request: outbound.OrderRequest{
			BaseQuantity: baseQuantity(price, follow.Order.BaseQuantity, follow.Order.QuoteQuantity),
			Price:        price,
		},
	})
	if err != nil {
		return err
	}

	updatedParent := updateParentEo(follow.Order, eo)
	modder.setParentOrder(updatedParent)
	return nil
}

func updateParentEo(o domain.ParentOrder, eo domain.ExchangeOrder) domain.ParentOrder {
	o.ExchangeOrder = eo
	o.Status = domain.EoStatusToOrderStatus(eo.Status())
	return o
}

func (f *follower) updateOrders(ctx context.Context, tick time.Time, followID string, ex outbound.Exchange) error {
	f.logger.Debug("updating parent order")
	err := f.modifyParentOrder(ctx, tick, followID, ex)
	if err != nil {
		return err
	}

	f.logger.Debug("updating stop orders")
	return f.modifyStops(ctx, tick, followID, ex)
}

var errOrderFinished = errors.New("order finished")

func (f *follower) modifyParentOrder(ctx context.Context, tick time.Time, followID string, ex outbound.Exchange) error {
	modder := f.registry.getModder(followID)
	follow := modder.getFollow()
	parent := follow.Order

	switch parent.Status {
	case domain.OrderStatusCanceled, domain.OrderStatusError:
		f.logger.Debug("1 parent order done, returning finished err")
		return errOrderFinished
	}

	eo, err := ex.GetOrder(ctx, parent.ExchangeOrder)
	if err != nil {
		return err
	}

	// Updated
	parent = updateParentEo(parent, eo)
	modder.setParentOrder(parent)

	switch parent.Status {
	case domain.OrderStatusActive:
		f.logger.Debug("2 parent order active, skipping update")
		return nil
	case domain.OrderStatusCanceled, domain.OrderStatusError:
		f.logger.Debug("2 parent order done, returning finished err")
		return errOrderFinished
	}

	price, err := parent.Plot.At(tick)
	if err != nil {
		return err
	}

	eo, err = ex.ModifyOrder(ctx, outbound.ModifyOrderRequest{
		ExchangeOrder: eo,
		Request: outbound.OrderRequest{
			BaseQuantity: baseQuantity(price, parent.BaseQuantity, parent.QuoteQuantity),
			Price:        price,
		},
	})
	if err != nil {
		return err
	}

	updatedParent := updateParentEo(follow.Order, eo)
	modder.setParentOrder(updatedParent)
	return nil
}

func (f *follower) modifyStops(ctx context.Context, tick time.Time, followID string, ex outbound.Exchange) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		return f.modifySLs(ctx, tick, followID, ex)
	})
	eg.Go(func() error {
		return f.modifyTPs(ctx, tick, followID, ex)
	})
	return eg.Wait()
}

func (f *follower) cancelStops(ctx context.Context, tick time.Time, followID string, ex outbound.Exchange) error {
	f.logger.Debug("updating take profits")
	wg := sync.WaitGroup{}
	stops := f.registry.getModder(followID).getStops()
	wg.Add(len(stops))
	for _, stop := range stops {
		go func(stop domain.StopOrder) {
			defer wg.Done()
			if err := ex.CancelOrder(ctx, stop.ExchangeOrder); err != nil {
				f.logger.Errorf("error cancelling order: %v", err)
			}
		}(stop)
	}
	wg.Wait()
	return nil
}

func (f *follower) modifyTPs(ctx context.Context, tick time.Time, followID string, ex outbound.Exchange) error {
	follow := f.registry.getFollow(followID)
	f.logger.Debug("updating take profits")
	eg := errgroup.Group{}
	for _, tp := range follow.TakeProfits {
		func(takeProfit domain.StopOrder) {
			eg.Go(func() error {
				return f.modifyTP(ctx, tick, followID, takeProfit.ID, ex)
			})
		}(tp)
	}
	return eg.Wait()
}

func (f *follower) modifyTP(ctx context.Context, tick time.Time, followID string, stopID string, ex outbound.Exchange) error {
	modder := f.registry.getModder(followID)
	tp := modder.getStop(stopID)

	logger := f.logger.Named(tp.ID)

	switch tp.Status {
	case domain.StopStatusCanceled, domain.StopStatusDone, domain.StopStatusError:
		logger.Debugf("TP order finished with status %s, returning", tp.Status)
		return nil //todo
	}

	eo, err := ex.GetOrder(ctx, tp.ExchangeOrder)
	if err != nil {
		return err
	}

	tp = updateStopEo(tp, eo)
	modder.setStop(tp)

	switch tp.Status {
	case domain.StopStatusCanceled, domain.StopStatusDone, domain.StopStatusError:
		logger.Debugf("TP order finished after refresh with status %s, returning", tp.Status)
		return nil //todo
	}

	if err := ex.CancelOrder(ctx, eo); err != nil {
		return err
	}

	price, err := tp.Plot.At(tick)
	if err != nil {
		return err
	}

	logger.Debug("creating new TP order")
	parent := modder.getParentOrder()
	eo, err = ex.CreateTakeProfitOrder(ctx, outbound.CreateTakeProfitRequest{
		Parent: parent.ExchangeOrder,
		Request: outbound.TakeProfitRequest{
			BaseQuantity: parent.ExchangeOrder.BaseQuantity() * tp.QuantityPct,
			Price:        0,
			StopPrice:    price,
		},
	})
	if err != nil {
		return err
	}

	logger.Debugf("new eo ID: %s", eo.ID())

	tp = updateStopEo(tp, eo)
	modder.setStop(tp)

	logger.Debugf("new eo updated ID: %s", tp.ExchangeOrder.ID())

	return nil
}

func (f *follower) modifySLs(ctx context.Context, tick time.Time, followID string, ex outbound.Exchange) error {
	eg := errgroup.Group{}
	follow := f.registry.getFollow(followID)
	for _, sl := range follow.StopLosses {
		func(stopLoss domain.StopOrder) {
			eg.Go(func() error {
				return f.modifySL(ctx, tick, followID, stopLoss.ID, ex)
			})
		}(sl)
	}
	return eg.Wait()
}

func (f *follower) modifySL(ctx context.Context, tick time.Time, followID string, stopID string, ex outbound.Exchange) error {
	modder := f.registry.getModder(followID)
	sl := modder.getStop(stopID)

	logger := f.logger.Named(sl.ID)

	switch sl.Status {
	case domain.StopStatusCanceled, domain.StopStatusDone, domain.StopStatusError:
		logger.Debug("TP order finished, returning")
		return nil //todo
	}

	eo, err := ex.GetOrder(ctx, sl.ExchangeOrder)
	if err != nil {
		return err
	}

	sl = updateStopEo(sl, eo)
	modder.setStop(sl)

	switch sl.Status {
	case domain.StopStatusCanceled, domain.StopStatusDone, domain.StopStatusError:
		logger.Debug("TP order finished, returning")
		return nil //todo
	}

	if err := ex.CancelOrder(ctx, eo); err != nil {
		return err
	}

	price, err := sl.Plot.At(tick)
	if err != nil {
		return err
	}

	parent := modder.getParentOrder()
	eo, err = ex.CreateStopLossOrder(ctx, outbound.CreateStopLossRequest{
		Parent: parent.ExchangeOrder,
		Request: outbound.StopLossRequest{
			BaseQuantity: parent.ExchangeOrder.BaseQuantity() * sl.QuantityPct,
			Price:        0,
			StopPrice:    price,
		},
	})
	if err != nil {
		return err
	}

	sl = updateStopEo(sl, eo)
	modder.setStop(sl)
	return nil
}

type intervalLoop struct {
	logger         *zap.SugaredLogger
	itv, headstart time.Duration
	stopC          chan struct{}
}

func newIntervalLoop(logger *zap.SugaredLogger, itv, headstart time.Duration) *intervalLoop {
	return &intervalLoop{
		logger:    logger,
		itv:       itv,
		headstart: headstart,
		stopC:     make(chan struct{}),
	}
}

func (l *intervalLoop) loop(fn func(time.Time) error) error {
	for {
		nextStart := NextIntervalStart(time.Now(), l.itv).Add(-l.headstart)
		l.logger.Debugf("next interval: %s", nextStart.String())

		select {
		case t := <-time.After(time.Until(nextStart)):
			err := fn(t.Add(l.headstart))
			if err != nil {
				return err
			}
		case <-l.stopC:
			return nil
		}
	}
}

func (f *follower) publishFollowUpdate(ctx context.Context, followID string) {
	go func() {
		err := f.publisher.PublishFollowUpdate(ctx, outbound.FollowUpdate{
			Follow: f.registry.getFollow(followID),
		})
		if err != nil {
			f.logger.Error(err)
		}
	}()
}

func baseQuantity(price, base, quote float64) float64 {
	if base != 0 {
		return base
	}

	return quote / price
}
