package followsvc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/inbound"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var validate = validator.New()

type Config struct {
	Logger     *zap.SugaredLogger
	Publisher  outbound.Publisher
	Repository outbound.Repository
}

type Service struct {
	logger    *zap.SugaredLogger
	loops     map[string]*intervalLoop
	publisher outbound.Publisher
	repo      outbound.Repository
	mu        *sync.Mutex
}

func New(cfg Config) *Service {

	return &Service{
		logger:    cfg.Logger,
		publisher: cfg.Publisher,
		loops:     map[string]*intervalLoop{},
		repo:      cfg.Repository,
		mu:        &sync.Mutex{},
	}
}

func (s *Service) CreateFollow(ctx context.Context, req inbound.CreateFollowRequest) (inbound.CreateFollowResponse, error) {
	return s.createFollow(ctx, req)
}

func (s *Service) GetFollow(ctx context.Context, req inbound.GetFollowRequest) (inbound.GetFollowResponse, error) {
	follow, err := s.repo.GetFollow(ctx, outbound.GetFollowRequest{
		FollowID: req.FollowID,
	})
	if err != nil {
		return inbound.GetFollowResponse{}, err
	}
	return inbound.GetFollowResponse{
		Follow: follow,
	}, nil
}

func (s *Service) StopFollow(ctx context.Context, req inbound.StopFollowRequest) error {
	return s.stopFollow(ctx, req)
}

func (s *Service) createFollow(ctx context.Context, req inbound.CreateFollowRequest) (inbound.CreateFollowResponse, error) {
	follow, orders, exchange, err := s.parseFollowReq(ctx, req)
	if err != nil {
		return inbound.CreateFollowResponse{}, err
	}

	if err := s.setupRepoFollow(ctx, follow, orders); err != nil {
		return inbound.CreateFollowResponse{}, err
	}

	handler := s.loopHandler(ctx, follow.ID, exchange)

	if err := handler(time.Now()); err != nil {
		return inbound.CreateFollowResponse{}, err
	}

	loop := s.newIntervalLoop(s.logger, follow.ID, follow.Interval, handler)

	go func() {
		defer func() {
			s.logger.Info("loop goroutine finished")
		}()
		if err := loop.loop(); err != nil {
			s.logger.Error(err)
			return
		}
	}()

	return inbound.CreateFollowResponse{
		FollowID: follow.ID,
	}, nil
}

func (s *Service) stopFollow(ctx context.Context, req inbound.StopFollowRequest) error {
	errs := []error{}
	exchange, err := parseExchange(s.logger, req.Exchange) //todo use hash to validate
	if err != nil {
		return err
	}

	err = s.stopLoop(req.FollowID)
	if err != nil {
		errs = append(errs, err)
	}

	follow, err := s.repo.GetFollow(ctx, outbound.GetFollowRequest{
		FollowID: req.FollowID,
	})
	if err != nil {
		errs = append(errs, err)
		return errors.Join(errs...)
	}

	follow.Status = domain.FollowStatusStopped
	err = s.repo.UpdateFollow(ctx, outbound.UpdateFollowRequest{
		Follow: follow,
	})
	if err != nil {
		errs = append(errs, err)
	}

	if !req.CancelOrders {
		return errors.Join(errs...)
	}

	orders := []domain.Order{}
	for _, orderID := range follow.OrderIDs {
		order, err := s.getOrder(ctx, orderID, exchange)
		if err != nil {
			errs = append(errs, err)
		}
		orders = append(orders, order)
	}

	return s.cancelOrders(ctx, orders, exchange)
}

func (s *Service) stopLoop(followID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	loop, ok := s.loops[followID]
	if !ok {
		return fmt.Errorf("follow %s not found in active loops", followID)
	}
	close(loop.stopC)
	return nil
}

// cancelOrders cancels all orders one by one sequentially and returns a joined error
func (s *Service) cancelOrders(ctx context.Context, orders []domain.Order, exchange outbound.Exchange) error {
	errs := []error{}
	for _, order := range orders {
		if order.ExchangeOrder == nil {
			continue
		}
		err := s.cancelOrder(ctx, order, exchange)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// cancelOrder cancels the ExchangeOrder and sets OrderStatus to CANCELED
// if ExchangeOrder is nil only the status is changed
func (s *Service) cancelOrder(ctx context.Context, order domain.Order, exchange outbound.Exchange) error {
	if order.ExchangeOrder == nil {
		order.Status = domain.OrderStatusCanceled
		return s.repo.UpdateOrder(ctx, outbound.UpdateOrderRequest{
			Order: order,
		})
	}

	canceledEO, err := exchange.CancelOrder(ctx, outbound.CancelExchangeOrdersRequest{
		EO: order.ExchangeOrder,
	})
	if err != nil {
		return err
	}

	order.ExchangeOrder = canceledEO
	return s.repo.UpdateOrder(ctx, outbound.UpdateOrderRequest{
		Order: order,
	})
}

func (s *Service) setupRepoFollow(ctx context.Context, follow domain.Follow, orders []domain.Order) (err error) {
	if err := s.repo.CreateFollow(ctx, outbound.CreateFollowRequest{
		Follow: follow,
	}); err != nil {
		return fmt.Errorf("error creating follow in repo: %v", err)
	}

	createdOrders := []domain.Order{}

	// todo check this out
	// defer func() {
	// 	if err == nil {
	// 		return
	// 	}
	// 	if e := s.cancelOrders(ctx, createdOrders); e != nil {
	// 		err = fmt.Errorf("%w: %w", err, fmt.Errorf("error canceling orders in cleanup: %w", e))
	// 	}
	// }()

	for _, order := range orders {
		if err := s.repo.CreateOrder(ctx, outbound.CreateOrderRequest{
			Order: order,
		}); err != nil {
			return fmt.Errorf("error creating order %+v in repo repo: %v", order, err)
		}
		createdOrders = append(createdOrders, order)
	}

	return nil
}

func (s *Service) loopHandler(ctx context.Context, followID string, exchange outbound.Exchange) func(t time.Time) error {
	return func(t time.Time) error {
		s.logger.Debug("running loop handler")
		follow, err := s.repo.GetFollow(ctx, outbound.GetFollowRequest{
			FollowID: followID,
		})
		if err != nil {
			return fmt.Errorf("error getting follow from repo: %v", err)
		}

		orders, err := s.getOrders(ctx, follow.OrderIDs, exchange)
		if err != nil {
			return fmt.Errorf("error getting follow orders: %v", err)
		}

		// create exchange orders
		created, err := s.createExchangeOrders(ctx, orders, exchange)
		orders = replaceOrders(orders, created)
		if err != nil {
			cancelErr := s.cancelOrders(ctx, orders, exchange)
			return fmt.Errorf("%w: %w", err, cancelErr)
		}

		// update echange orders
		ordersToModify := slices.DeleteFunc(slices.Clone(orders), func(o domain.Order) bool {
			return slices.ContainsFunc(created, func(created domain.Order) bool {
				return created.ID == o.ID
			})
		})

		modifiedOrders, err := s.modifyExchangeOrders(ctx, ordersToModify, exchange)
		orders = replaceOrders(orders, modifiedOrders)
		if err != nil {
			cancelErr := s.cancelOrders(ctx, orders, exchange)
			return fmt.Errorf("%w: %w", err, cancelErr)
		}

		return s.publisher.PublishFollowUpdate(ctx, outbound.FollowUpdate{
			Follow: follow,
		})
	}
}

func (s *Service) createExchangeOrders(ctx context.Context, orders []domain.Order, exchange outbound.Exchange) ([]domain.Order, error) {
	created := []domain.Order{}
	for _, order := range orders {
		if order.ExchangeOrder != nil && order.ExchangeOrder.Status != "" {
			continue
		}
		order, err := s.createExchangeOrder(ctx, order, exchange)
		if err != nil {
			return created, err
		}
		created = append(created, order)
		if err := s.repo.UpdateOrder(ctx, outbound.UpdateOrderRequest{
			Order: order,
		}); err != nil {
			return created, err
		}

	}
	return created, nil
}

func (s *Service) createExchangeOrder(ctx context.Context, order domain.Order, exchange outbound.Exchange) (domain.Order, error) {
	price, err := order.Plot.At(time.Now())
	if err != nil {
		return domain.Order{}, nil
	}

	eo, err := exchange.CreateOrder(ctx, outbound.CreateExchangeOrderRequest{
		Pair:         order.Pair,
		Type:         order.Type,
		Side:         order.Side,
		BaseQuantity: baseQuantity(price, order.BaseQuantity, order.QuoteQuantity),
		Price:        price,
		StopPrice:    0,
	})
	if err != nil {
		return domain.Order{}, err
	}

	order.ExchangeOrder = eo
	return order, nil
}

func (s *Service) modifyExchangeOrders(ctx context.Context, orders []domain.Order, exchange outbound.Exchange) ([]domain.Order, error) {
	modified := []domain.Order{}
	for _, order := range orders {
		if order.ExchangeOrder == nil {
			return modified, fmt.Errorf("unexpected nil exchange order for order %s", order.ID)
		}
		order, err := s.modifyExchangeOrder(ctx, order, exchange)
		if err != nil {
			return modified, err
		}
		modified = append(modified, order)
		if err := s.repo.UpdateOrder(ctx, outbound.UpdateOrderRequest{
			Order: order,
		}); err != nil {
			return modified, err
		}
	}
	return modified, nil
}

func (s *Service) modifyExchangeOrder(ctx context.Context, order domain.Order, exchange outbound.Exchange) (domain.Order, error) {
	price, err := order.Plot.At(time.Now())
	if err != nil {
		return domain.Order{}, nil
	}

	eo, err := exchange.ModifyOrder(ctx, outbound.ModifyExchangeOrderRequest{
		EO:           order.ExchangeOrder,
		BaseQuantity: baseQuantity(price, order.BaseQuantity, order.QuoteQuantity),
		Price:        price,
		StopPrice:    0,
	})
	if err != nil {
		return domain.Order{}, nil
	}

	order.ExchangeOrder = eo
	return order, err
}

func (s *Service) getOrders(ctx context.Context, orderIDs []string, exchange outbound.Exchange) ([]domain.Order, error) {
	orders := []domain.Order{}
	for _, orderID := range orderIDs {
		order, err := s.getOrder(ctx, orderID, exchange)
		if err != nil {
			return orders, nil
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (s *Service) getOrder(ctx context.Context, orderID string, exchange outbound.Exchange) (domain.Order, error) {
	order, err := s.repo.GetOrder(ctx, outbound.GetOrderRequest{
		OrderID: orderID,
	})
	if err != nil {
		return domain.Order{}, nil
	}
	plot, err := order.PlotSpec.Parse()
	if err != nil {
		return domain.Order{}, nil
	}
	order.Plot = plot
	return order, nil
}

func (s *Service) newIntervalLoop(logger *zap.SugaredLogger, followID string, interval time.Duration, f func(time.Time) error) *intervalLoop {
	loop := newIntervalLoop(logger, interval, f)
	s.addLoop(followID, loop)
	return loop
}

func (s *Service) addLoop(followID string, loop *intervalLoop) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loops[followID] = loop
}

func (s *Service) parseFollowReq(ctx context.Context, req inbound.CreateFollowRequest) (domain.Follow, []domain.Order, outbound.Exchange, error) {
	if err := validate.Struct(req); err != nil {
		return domain.Follow{}, nil, nil, err
	}

	exchange, err := parseExchange(s.logger, req.Exchange)
	if err != nil {
		return domain.Follow{}, nil, nil, fmt.Errorf("error parsing exchange: %v", err)
	}

	if err := exchange.Init(ctx); err != nil {
		return domain.Follow{}, nil, nil, err
	}

	pair, err := parsePair(req.Symbol)
	if err != nil {
		return domain.Follow{}, nil, nil, err
	}

	interval, err := parseInterval(req.Interval)
	if err != nil {
		return domain.Follow{}, nil, nil, err
	}

	var orderIDs []string
	var orders []domain.Order
	for _, cro := range req.Orders {
		plot, err := cro.PlotSpec.Parse()
		if err != nil {
			return domain.Follow{}, nil, nil, fmt.Errorf("error parsing plot %+v: %w", cro.PlotSpec, err)
		}
		eHash, err := domain.Hash(req.Exchange)
		if err != nil {
			return domain.Follow{}, nil, nil, fmt.Errorf("hashing exchange: %w", err)
		}
		order := domain.Order{
			ID:            uuid.NewString(),
			Name:          cro.Name,
			Status:        domain.OrderStatusProcessing,
			Type:          cro.Type,
			Pair:          pair,
			Side:          cro.Side,
			QuoteQuantity: cro.QuoteQuantity,
			BaseQuantity:  cro.BaseQuantity,
			ClosePosition: cro.ClosePosition,
			ReduceOnly:    cro.ClosePosition,
			Relations:     cro.Relations,
			PlotSpec:      cro.PlotSpec,
			Plot:          plot,
			ExchangeHash:  eHash,
			ExchangeOrder: nil,
		}

		orders = append(orders, order)
		orderIDs = append(orderIDs, order.ID)
	}

	hash, err := domain.Hash(req.Exchange)
	if err != nil {
		return domain.Follow{}, nil, nil, fmt.Errorf("error calculating exchange hash: %v", err)
	}

	follow := domain.Follow{
		ID:           uuid.NewString(),
		Status:       domain.FollowStatusPending,
		ExchangeHash: hash,
		Pair:         pair,
		Interval:     interval,
		WebhookURL:   req.WebhookURL,
		OrderIDs:     orderIDs,
	}
	if err := validate.Struct(follow); err != nil {
		return domain.Follow{}, nil, nil, err
	}

	return follow, orders, exchange, nil
}
