package followsvc

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
	logger   *zap.SugaredLogger
	follower *follower
}

func New(cfg Config) *Service {
	return &Service{
		logger:   cfg.Logger,
		follower: newFollower(cfg.Logger, cfg.Pubblisher),
	}
}

func (s *Service) StartFollow(ctx context.Context, req inbound.CreateFollowRequest) error {
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

	orderPlot, err := parsePlotMap(req.Order.Plot)
	if err != nil {
		return errors.Wrap(err, "error parsing price plot")
	}

	var takeProfits, stopLosses []domain.StopOrder
	for _, tp := range req.TakeProfits {
		plot, err := parsePlotMap(tp.Plot)
		if err != nil {
			return errors.Wrap(err, "error parsing take profit plot")
		}
		takeProfits = append(takeProfits, domain.StopOrder{
			ID:          uuid.NewString(),
			Status:      domain.StopStatusPending,
			QuantityPct: tp.QuantityPct,
			Plot:        plot,
		})
	}
	for _, sl := range req.StopLosses {
		plot, err := parsePlotMap(sl.Plot)
		if err != nil {
			return errors.Wrap(err, "error parsing stop loss plot")
		}
		stopLosses = append(stopLosses, domain.StopOrder{
			ID:          uuid.NewString(),
			Status:      domain.StopStatusPending,
			QuantityPct: sl.QuantityPct,
			Plot:        plot,
		})
	}

	follow := domain.Follow{
		ID:           uuid.NewString(),
		Exchange:     req.Exchange.Name,
		Pair:         pair,
		PositionSide: domain.PositionSide(req.Side),
		Interval:     interval,
		WebhookURL:   req.Webhook,

		Order: domain.ParentOrder{
			ID:            uuid.NewString(),
			Status:        domain.OrderStatusPending,
			BaseQuantity:  req.Order.BaseQuantity,
			QuoteQuantity: req.Order.QuoteQuantity,
			Plot:          orderPlot,
		},
		TakeProfits: takeProfits,
		StopLosses:  stopLosses,
	}

	if err := validate.Struct(follow); err != nil {
		return err
	}

	return s.follower.startFollow(ctx, follow, exchange)
}

func (s *Service) StopFollow(ctx context.Context, req inbound.CancelFollowRequest) error {
	return nil ///return s.updater.stopFollow(req.FollowID)
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
