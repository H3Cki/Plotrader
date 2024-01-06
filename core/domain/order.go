package domain

import (
	"errors"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
)

type PositionSide string
type StopType string
type StopStatus string
type OrderStatus string
type ExchangeOrderStatus string

var (
	ErrFollowNotFound = errors.New("follow not found")

	PositionSideLong  PositionSide = "LONG"
	PositionSideShort PositionSide = "SHORT"

	OrderStatusPending  OrderStatus = "PENDING"
	OrderStatusOpen     OrderStatus = "OPEN"
	OrderStatusActive   OrderStatus = "ACTIVE"
	OrderStatusCanceled OrderStatus = "CANCELED"
	OrderStatusError    OrderStatus = "ERROR"

	StopStatusPending  StopStatus = "PENDING"
	StopStatusOpen     StopStatus = "OPEN"
	StopStatusDone     StopStatus = "DONE"
	StopStatusCanceled StopStatus = "CANCELED"
	StopStatusError    StopStatus = "ERROR"

	ExchangeOrderStatusPending         ExchangeOrderStatus = "PENDING"
	ExchangeOrderStatusOpen            ExchangeOrderStatus = "OPEN"
	ExchangeOrderStatusFilled          ExchangeOrderStatus = "FILLED"
	ExchangeOrderStatusPartiallyFilled ExchangeOrderStatus = "PARTIALLY_FILLED"
	ExchangeOrerStatusCanceled         ExchangeOrderStatus = "CANCELED"
)

func EoStatusToOrderStatus(eos ExchangeOrderStatus) OrderStatus {
	switch eos {
	case ExchangeOrderStatusPending:
		return OrderStatusPending
	case ExchangeOrderStatusOpen:
		return OrderStatusOpen
	case ExchangeOrderStatusFilled, ExchangeOrderStatusPartiallyFilled:
		return OrderStatusActive
	case ExchangeOrerStatusCanceled:
		return OrderStatusCanceled
	}
	return ""
}

func EoStatusToStopStatus(eos ExchangeOrderStatus) StopStatus {
	switch eos {
	case ExchangeOrderStatusPending:
		return StopStatusPending
	case ExchangeOrderStatusOpen:
		return StopStatusOpen
	case ExchangeOrderStatusFilled, ExchangeOrderStatusPartiallyFilled:
		return StopStatusDone
	case ExchangeOrerStatusCanceled:
		return StopStatusCanceled
	}
	return ""
}

type Follow struct {
	ID           string
	Exchange     string
	Pair         Pair
	PositionSide PositionSide
	Interval     time.Duration
	LastTick     time.Time
	WebhookURL   string

	Order       ParentOrder
	TakeProfits []StopOrder
	StopLosses  []StopOrder
}

type Pair struct {
	Base, Quote string
}

type ParentOrder struct {
	ID            string        `json:"id"`
	Status        OrderStatus   `json:"status"`
	ExchangeOrder ExchangeOrder `json:"exchangeOrder"`
	QuoteQuantity float64       `json:"quoteQuantity"`
	BaseQuantity  float64       `json:"baseQuantity"`
	Plot          geometry.Plot `json:"plot"`
}

type StopOrder struct {
	ID            string        `json:"id"`
	Status        StopStatus    `json:"status"`
	ExchangeOrder ExchangeOrder `json:"exchangeOrder"`
	QuantityPct   float64       `json:"quantityPct"`
	Plot          geometry.Plot `json:"plot"`
}

type ExchangeOrder interface {
	ID() string
	Status() ExchangeOrderStatus
	CreatedAt() time.Time
	Price() float64
	StopPrice() float64
	BaseQuantity() float64
}
