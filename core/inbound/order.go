package inbound

import (
	"context"
	"encoding/json"

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

type UpdaterService interface {
	CreateOrder(context.Context, CreateOrderRequest) error
	CancelOrder(context.Context, CancelOrderRequest) error
}

type CreateOrderRequest struct {
	Exchange ExchangeConfig `json:"exchange" validate:"required"`
	Symbol   string         `json:"symbol" validate:"required"`
	Side     string         `json:"side" validate:"required"`

	Interval          string `json:"interval" validate:"required"`
	DisableProtection bool   `json:"disableProtection"`
	WaitNextInterval  bool   `json:"waitNextInterval"`

	Order      OrderRequest `json:"order" validate:"required"`
	TakeProfit *StopRequest `json:"takeProfit"`
	StopLoss   *StopRequest `json:"stopLoss"`
}

type OrderRequest struct {
	Type          string         `json:"type" validate:"required"`
	TimeInForce   string         `json:"timeInForce" validate:"required"`
	QuoteQuantity float64        `json:"quoteQuantity"`
	BaseQuantity  float64        `json:"baseQuantity"`
	Plot          map[string]any `json:"plot" validate:"required"`
}

type StopRequest struct {
	Type        string         `json:"type" validate:"required"`
	TimeInForce string         `json:"timeInForce" validate:"required"`
	QuantityPct float64        `json:"quantityPct" alidate:"required"`
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

type CancelOrderRequest struct {
	OrderID string
}
