package domain

import (
	"errors"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
)

type PositionSide string

var (
	ErrOrderNotFound = errors.New("order not found")
)

var (
	PositionSideLong  PositionSide = "LONG"
	PositionSideShort PositionSide = "SHORT"
)

type Follow struct {
	ID           string        `json:"id"`
	Exchange     string        `json:"exchange"`
	Pair         Pair          `json:"symbol"`
	PositionSide PositionSide  `json:"side"`
	Interval     time.Duration `json:"interval"`
	WebhookURL   string        `json:"webhook"`
	Orders       Orders        `json:"orders"`
}
type Pair struct {
	Base, Quote string
}

type Orders struct {
	Order       Order
	TakeProfits []TPSLOrder
	StopLosses  []TPSLOrder
}

type Order struct {
	ExchangeOrder ExchangeOrder `json:"exchangeOrder"`
	QuoteQuantity float64       `json:"quoteQuantity"`
	BaseQuantity  float64       `json:"baseQuantity"`
	Plot          geometry.Plot `json:"plot" validate:"required"`
}

type TPSLOrder struct {
	ExchangeOrder ExchangeOrder `json:"exchangeOrder"`
	QuantityPct   float64       `json:"quantityPct" alidate:"required"`
	Plot          geometry.Plot `json:"plot" validate:"required"`
}

type ExchangeOrderStatus string

var (
	ExchangeOrderStatusOpen            ExchangeOrderStatus = "OPEN"
	ExchangeOrderStatusFilled          ExchangeOrderStatus = "FILLED"
	ExchangeOrderStatusPartiallyFilled ExchangeOrderStatus = "PARTIALLY_FILLED"
	ExchangeOrerStatusCanceled         ExchangeOrderStatus = "CANCELED"
)

type ExchangeOrder interface {
	Status() ExchangeOrderStatus
	CreatedAt() time.Time
	Price() float64
	StopPrice() float64
	BaseQuantity() float64
}
