package domain

import (
	"errors"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
)

type OrderSide string
type OrderType string
type TimeInForce string
type OrderStatus string

var (
	ErrOrderNotFound = errors.New("order not found")
)

var (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"

	OrderTypeLimit      OrderType = "limit"
	OrderTypeTakeProfit OrderType = "takeProfit"
	OrderTypeStopLoss   OrderType = "stopLoss"

	TimeInForceGTC TimeInForce = "gtc"

	OrderStatusPending  OrderStatus = "pending"
	OrderStatusCreated  OrderStatus = "created"
	OrderStatusActive   OrderStatus = "active"
	OrderStatusFinished OrderStatus = "finished"
	OrderStatusError    OrderStatus = "error"
	OrderStatusCanceled OrderStatus = "canceled"
)

type Order struct {
	ID             string
	Status         OrderStatus
	Exchange       string
	Params         OrderParams
	ExchangeOrders []ExchangeOrders
}

type OrderParams struct {
	Pair     Pair          `json:"symbol" validate:"required"`
	Side     OrderSide     `json:"side" validate:"required"`
	Interval time.Duration `json:"interval" validate:"required"`

	DisableProtection bool `json:"protection"`
	WaitNextInterval  bool `json:"waitNextInterval"`

	Order      OrderRequest `json:"order" validate:"required"`
	TakeProfit *StopRequest `json:"takeProfitPlot"`
	StopLoss   *StopRequest `json:"stopLossPlot"`

	WebhookURL string `json:"webhook"`
}

type Pair struct {
	Base, Quote string
}

type OrderRequest struct {
	Type          OrderType     `json:"type" validate:"required"`
	TimeInForce   TimeInForce   `json:"timeInForce" validate:"required"`
	QuoteQuantity float64       `json:"quoteQuantity"`
	BaseQuantity  float64       `json:"baseQuantity"`
	Plot          geometry.Plot `json:"plot" validate:"required"`
}

type StopRequest struct {
	Type        OrderType     `json:"type" validate:"required"`
	TimeInForce TimeInForce   `json:"timeInForce" validate:"required"`
	QuantityPct float64       `json:"quantityPct" alidate:"required"`
	Plot        geometry.Plot `json:"plot" validate:"required"`
}

type ExchangeOrders struct {
	Time       time.Time     `json:"time"`
	Order      ExchangeOrder `json:"order"`
	TakeProfit ExchangeOrder `json:"takeProfit,omitempty"`
	StopLoss   ExchangeOrder `json:"stopLoss,omitempty"`
}

type ExchangeOrder struct {
	ID           string      `json:"id"`
	Status       OrderStatus `json:"status"`
	Message      string      `json:"message"`
	Price        float64     `json:"price"`
	BaseQuantity float64     `json:"baseQuantity"`
}
