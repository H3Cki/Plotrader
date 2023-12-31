package updatersvc

import (
	"context"
	"strings"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/inbound"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var validate = validator.New()

type Config struct {
	Logger     *zap.SugaredLogger
	Pubblisher outbound.Publisher
}

type Service struct {
	logger  *zap.SugaredLogger
	updater *updater
}

func New(cfg Config) *Service {
	return &Service{
		logger:  cfg.Logger,
		updater: newFollower(cfg.Logger, cfg.Pubblisher),
	}
}

func (s *Service) CreateOrder(ctx context.Context, req inbound.CreateOrderRequest) error {
	s.logger.Debug("creating order")

	if err := validate.Struct(req); err != nil {
		return err
	}

	exchange, err := parseExchange(s.logger, req.Exchange)
	if err != nil {
		return errors.Wrap(err, "error parsing exchange")
	}

	if err := exchange.Init(ctx); err != nil {
		return err
	}

	pair, err := parsePair(req.Symbol)
	if err != nil {
		return err
	}

	interval, err := parseInterval(req.Interval)
	if err != nil {
		return err
	}

	orderPlot, err := parsePlotMap(req.Order.Plot, !req.DisableProtection)
	if err != nil {
		return errors.Wrap(err, "error parsing price plot")
	}

	var takeProfit, stopLoss *domain.StopRequest
	if req.TakeProfit != nil {
		tp, err := parsePlotMap(req.TakeProfit.Plot, !req.DisableProtection)
		if err != nil {
			return errors.Wrap(err, "error parsing take profit plot")
		}
		takeProfit = &domain.StopRequest{
			Type:        domain.OrderType(req.TakeProfit.Type),
			TimeInForce: domain.TimeInForce(req.TakeProfit.TimeInForce),
			QuantityPct: req.TakeProfit.QuantityPct,
			Plot:        tp,
		}
	}

	if req.StopLoss != nil {
		sl, err := parsePlotMap(req.StopLoss.Plot, !req.DisableProtection)
		if err != nil {
			return errors.Wrap(err, "error parsing stop loss plot")
		}
		stopLoss = &domain.StopRequest{
			Type:        domain.OrderType(req.StopLoss.Type),
			TimeInForce: domain.TimeInForce(req.StopLoss.TimeInForce),
			QuantityPct: req.StopLoss.QuantityPct,
			Plot:        sl,
		}
	}

	order := &domain.Order{
		ID:       uuid.NewString(),
		Status:   domain.OrderStatusPending,
		Exchange: req.Exchange.Name,
		Params: domain.OrderParams{
			Pair:              pair,
			Side:              domain.OrderSide(req.Side),
			Interval:          interval,
			DisableProtection: req.DisableProtection,
			WaitNextInterval:  req.WaitNextInterval,
			Order: domain.OrderRequest{
				Type:          domain.OrderType(req.Order.Type),
				TimeInForce:   domain.TimeInForce(req.Order.TimeInForce),
				QuoteQuantity: req.Order.QuoteQuantity,
				BaseQuantity:  req.Order.BaseQuantity,
				Plot:          orderPlot,
			},
			TakeProfit: takeProfit,
			StopLoss:   stopLoss,
		},
		ExchangeOrders: []domain.ExchangeOrders{},
	}

	if err := validate.Struct(order); err != nil {
		return err
	}

	return s.updater.createOrder(ctx, order, exchange)
}

func (s *Service) CancelOrder(ctx context.Context, req inbound.CancelOrderRequest) error {
	return s.updater.cancelOrder(req.OrderID)
}

var predefinedDurations = map[string]time.Duration{
	"1d": 24 * time.Hour,
	"2d": 2 * 24 * time.Hour,
	"3d": 3 * 24 * time.Hour,
	"4d": 4 * 24 * time.Hour,
	"5d": 5 * 24 * time.Hour,
	"6d": 6 * 24 * time.Hour,
	"1w": 7 * 24 * time.Hour,
	"2w": 14 * 24 * time.Hour,
	"1M": 30 * 24 * time.Hour,
}

// parseInterval adds more units on top of time.Parse():
// 1d, 2d, 3d, 4d, 5d, 6d, 1w, 2w, 1M.
// These units are added for convenience and cannot be combined e.g. Parse("1d12h") or Parse("1w1d") wont work.
func parseInterval(itv string) (time.Duration, error) {
	if d, ok := predefinedDurations[itv]; ok {
		return d, nil
	}

	d, err := time.ParseDuration(itv)
	if err != nil {
		return 0, err
	}

	return d, nil
}

func parsePair(s string) (domain.Pair, error) {
	bq := strings.Split(s, "-")
	if len(bq) != 2 {
		return domain.Pair{}, errors.New("invalid symbol")
	}

	return domain.Pair{
		Base:  bq[0],
		Quote: bq[1],
	}, nil
}
