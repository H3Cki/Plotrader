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
	mu        sync.Mutex
}

func newFollower(logger *zap.SugaredLogger, publisher outbound.Publisher) *follower {
	return &follower{
		logger:    logger,
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
	fm := newManager(&follow)

	f.logger.Debug("creating orders")

	if err := f.createOrders(ctx, fm, exchange); err != nil {
		return err
	}

	f.publishFollowUpdate(ctx, fm.follow())

	loop := f.newIntervalLoop(follow.ID, follow.Interval, 500*time.Millisecond)
	go func() {
		loop.loop(func(tick time.Time) error {
			err := f.handleTick(ctx, tick, fm, exchange)
			if err != nil {
				f.logger.Errorf("error handling interval: %v", err)
			}
			return err
		})
	}()

	return nil
}

func (f *follower) handleTick(ctx context.Context, tick time.Time, fm *followManager, ex outbound.Exchange) error {
	f.logger.Debugf("handling tick at %s", tick.String())

	uCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	defer func() {
		f.publishFollowUpdate(ctx, fm.follow())
	}()

	err := f.updateOrders(uCtx, tick, fm, ex)

	if err != nil {
		f.logger.Debug("cancelling all orders due to error")
		if cErr := f.cancelOrders(ctx, fm, ex); cErr != nil {
			err = fmt.Errorf("%w : %w", err, cErr)
		}
	}

	return err
}

func (f *follower) cancelOrders(ctx context.Context, fm *followManager, ex outbound.Exchange) error {
	eos := []domain.ExchangeOrder{fm.getOrder().ExchangeOrder}
	for _, tp := range fm.getTPs() {
		eos = append(eos, tp.ExchangeOrder)
	}
	for _, sl := range fm.getSLs() {
		eos = append(eos, sl.ExchangeOrder)
	}
	errs := []error{}
	for _, eo := range eos {
		errs = append(errs, ex.CancelOrder(ctx, eo))
	}
	return errors.Join(errs...)
}

func (f *follower) createOrders(ctx context.Context, fm *followManager, ex outbound.Exchange) error {
	follow := fm.follow()
	itvStart := IntervalStart(time.Now(), follow.Interval)

	if err := f.createParentOrder(ctx, itvStart, fm, ex); err != nil {
		return err
	}

	return f.createStops(ctx, itvStart, fm, ex)
}

func (f *follower) createStops(ctx context.Context, t time.Time, fm *followManager, ex outbound.Exchange) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		return f.createSLs(ctx, t, fm, ex)
	})
	eg.Go(func() error {
		return f.createTPs(ctx, t, fm, ex)
	})
	return eg.Wait()
}

func (f *follower) createTPs(ctx context.Context, t time.Time, fm *followManager, ex outbound.Exchange) error {
	eg := errgroup.Group{}
	for i, tp := range fm.getTPs() {
		func(idx int, tp domain.StopOrder) {
			eg.Go(func() error {
				price, err := tp.Plot.At(t)
				if err != nil {
					return err
				}
				order := fm.getOrder()
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
				status := domain.EoStatusToStopStatus(eo.Status())
				fm.updateTP(idx, withStopEo(eo), withStopStatus(status))
				return nil
			})
		}(i, tp)
	}
	return eg.Wait()
}

func (f *follower) createSLs(ctx context.Context, t time.Time, fm *followManager, ex outbound.Exchange) error {
	eg := errgroup.Group{}
	for i, sl := range fm.getSLs() {
		func(idx int, tp domain.StopOrder) {
			eg.Go(func() error {
				price, err := tp.Plot.At(t)
				if err != nil {
					return err
				}
				order := fm.getOrder()
				eo, err := ex.CreateStopLossOrder(ctx, outbound.CreateStopLossRequest{
					Parent: order.ExchangeOrder,
					Request: outbound.StopLossRequest{
						BaseQuantity: order.ExchangeOrder.BaseQuantity() * tp.QuantityPct,
						Price:        0,
						StopPrice:    price,
					},
				})
				if err != nil {
					return err
				}
				status := domain.EoStatusToStopStatus(eo.Status())
				fm.updateSL(idx, withStopEo(eo), withStopStatus(status))
				return nil
			})
		}(i, sl)
	}
	return eg.Wait()
}

func (f *follower) createParentOrder(ctx context.Context, t time.Time, fm *followManager, ex outbound.Exchange) error {
	follow := fm.follow()
	price, err := follow.Order.Plot.At(t)
	if err != nil {
		return err
	}

	f.logger.Debug("creating order on exchange")
	eo, err := ex.CreateOrder(ctx, outbound.CreateOrderRequest{
		Pair:    follow.Pair,
		PosSide: follow.PositionSide,
		Request: outbound.OrderRequest{
			BaseQuantity: baseQuantity(price, follow.Order.BaseQuantity, follow.Order.QuoteQuantity),
			Price:        price,
		},
	})
	if err != nil {
		return err
	}
	f.logger.Debug("updating parent status")
	fm.updateOrder(withOrderEo(eo), withOrderStatus(domain.EoStatusToOrderStatus(eo.Status())))
	return nil
}

func (f *follower) updateOrders(ctx context.Context, tick time.Time, fm *followManager, ex outbound.Exchange) error {
	f.logger.Debug("updating parent order")
	err := f.modifyParentOrder(ctx, tick, fm, ex)
	if err != nil {
		return err
	}

	f.logger.Debug("updating stop orders")
	return f.modifyStops(ctx, tick, fm, ex)
}

var errOrderFinished = errors.New("order finished")

func (f *follower) modifyParentOrder(ctx context.Context, tick time.Time, fm *followManager, ex outbound.Exchange) error {
	order := fm.getOrder()

	switch fm.getOrder().Status {
	case domain.OrderStatusActive, domain.OrderStatusCanceled, domain.OrderStatusError:
		f.logger.Debug("parent order finished, returning error")
		return errOrderFinished
	}

	f.logger.Debug("getting parent order")
	eo, err := ex.GetOrder(ctx, order.ExchangeOrder)
	if err != nil {
		return err
	}

	status := domain.EoStatusToOrderStatus(eo.Status())
	fm.updateOrder(withOrderEo(eo), withOrderStatus(status))

	switch status {
	case domain.OrderStatusActive, domain.OrderStatusCanceled, domain.OrderStatusError:
		f.logger.Debug("parent order finished after refresh, returning error")
		return errOrderFinished
	}

	price, err := order.Plot.At(tick)
	if err != nil {
		return err
	}

	f.logger.Debug("modifying parent order on exchange")
	eo, err = ex.ModifyOrder(ctx, outbound.ModifyOrderRequest{
		ExchangeOrder: eo,
		Request: outbound.OrderRequest{
			BaseQuantity: baseQuantity(price, order.BaseQuantity, order.QuoteQuantity),
			Price:        price,
		},
	})
	if err != nil {
		return err
	}

	f.logger.Debug("updating parent status")
	status = domain.EoStatusToOrderStatus(eo.Status())
	fm.updateOrder(withOrderEo(eo), withOrderStatus(status))
	return nil
}

func (f *follower) modifyStops(ctx context.Context, tick time.Time, fm *followManager, ex outbound.Exchange) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		return f.modifySLs(ctx, tick, fm, ex)
	})
	eg.Go(func() error {
		return f.modifyTPs(ctx, tick, fm, ex)
	})
	return eg.Wait()
}

func (f *follower) modifyTPs(ctx context.Context, tick time.Time, fm *followManager, ex outbound.Exchange) error {
	f.logger.Debug("updating take profits")
	eg := errgroup.Group{}
	for i, tp := range fm.getTPs() {
		func(idx int, tp domain.StopOrder) {
			eg.Go(func() error {
				return f.modifyTP(ctx, tick, fm, ex, i, tp)
			})
		}(i, tp)
	}
	return eg.Wait()
}

func (f *follower) modifyTP(ctx context.Context, tick time.Time, fm *followManager, ex outbound.Exchange, idx int, tp domain.StopOrder) error {
	logger := f.logger.Named(tp.ID)

	switch tp.Status {
	case domain.StopStatusCanceled, domain.StopStatusDone, domain.StopStatusError:
		logger.Debug("TP order finished, returning")
		return nil //todo
	}

	eo, err := ex.GetOrder(ctx, tp.ExchangeOrder)
	if err != nil {
		return err
	}

	status := domain.EoStatusToStopStatus(eo.Status())
	fm.updateTP(idx, withStopEo(eo), withStopStatus(status))

	switch status {
	case domain.StopStatusCanceled, domain.StopStatusDone, domain.StopStatusError:
		logger.Debug("TP order finished, returning")
		return nil //todo
	}

	logger.Debug("cancelling TP order")
	if err := ex.CancelOrder(ctx, eo); err != nil {
		return err
	}

	price, err := tp.Plot.At(tick)
	if err != nil {
		return err
	}

	logger.Debug("creating new TP order")
	order := fm.getOrder()
	eo, err = ex.CreateTakeProfitOrder(ctx, outbound.CreateTakeProfitRequest{
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

	logger.Debug("creating TP status")
	fm.updateTP(idx, withStopEo(eo), withStopStatus(status))
	return nil
}

func (f *follower) modifySLs(ctx context.Context, tick time.Time, fm *followManager, ex outbound.Exchange) error {
	eg := errgroup.Group{}
	for i, tp := range fm.getSLs() {
		func(idx int, tp domain.StopOrder) {
			eg.Go(func() error {
				return f.modifySL(ctx, tick, fm, ex, i, tp)
			})
		}(i, tp)
	}
	return eg.Wait()
}

func (f *follower) modifySL(ctx context.Context, tick time.Time, fm *followManager, ex outbound.Exchange, idx int, sl domain.StopOrder) error {
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

	status := domain.EoStatusToStopStatus(eo.Status())
	fm.updateSL(idx, withStopEo(eo), withStopStatus(status))

	switch status {
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

	order := fm.getOrder()
	eo, err = ex.CreateStopLossOrder(ctx, outbound.CreateStopLossRequest{
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

	fm.updateSL(idx, withStopEo(eo), withStopStatus(status))

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

func (f *follower) publishFollowUpdate(ctx context.Context, update domain.Follow) {
	go func() {
		err := f.publisher.PublishFollowUpdate(ctx, outbound.FollowUpdate{
			Follow: update,
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
