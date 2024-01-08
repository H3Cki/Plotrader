package inbound

import (
	"context"
	"encoding/json"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/go-playground/validator/v10"
)

var (
	validate = validator.New()
)

func init() {
	// 	validate.RegisterStructValidation(func(sl validator.StructLevel) {
	// 		e := sl.Current().Interface().(Exchange)

	// }, Exchange{})
}

type Action string

const (
	ActionCreateOrder Action = "create"
	ActionCancelOrder Action = "cancel"
)

type FollowerService interface {
	StartFollow(context.Context, CreateFollowRequest) error
	StopFollow(context.Context, CancelFollowRequest) error
}

type CreateFollowRequest struct {
	Exchange ExchangeConfig `json:"exchange" validate:"required"`
	Symbol   string         `json:"symbol" validate:"required"`
	Side     domain.Side    `json:"side" validate:"required"`
	Interval string         `json:"interval" validate:"required"`

	Order       OrderRequest  `json:"order" validate:"required"`
	TakeProfits []StopRequest `json:"takeProfits"`
	StopLosses  []StopRequest `json:"stopLosses"`

	Webhook string `json:"webhook"`
}

type OrderRequest struct {
	BaseQuantity  float64        `json:"baseQuantity"`
	QuoteQuantity float64        `json:"quoteQuantity"`
	Plot          map[string]any `json:"plot" validate:"required"`
}

type StopRequest struct {
	QuantityPct float64        `json:"quantityPct"`
	Plot        map[string]any `json:"plot" validate:"required"`
}

type ExchangeConfig struct {
	Name      string         `json:"name"`
	ConfigEnv string         `json:"configEnv"`
	Config    map[string]any `json:"config"`
}

func (e *ExchangeConfig) MarshalConfig(to any) error {
	bytes, err := json.Marshal(e.Config)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, to)
}

type CancelFollowRequest struct {
	FollowID string
}
