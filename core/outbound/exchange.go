package outbound

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
)

type Exchange interface {
	Init(context.Context) error
	GetPrice(context.Context, GetPriceRequest) (float64, error)
	CreateOrder(context.Context, CreateOrderRequest) (domain.ExchangeOrders, error)
	CancelOrder(context.Context, CancelOrderRequest) error
}

type GetPriceRequest struct {
	Pair domain.Pair
}

type CreateOrderRequest struct {
	Pair       domain.Pair
	Side       domain.OrderSide
	Order      OrderDetails
	TakeProfit *StopDetails
	StopLoss   *StopDetails
}

type OrderDetails struct {
	Type         domain.OrderType
	TimeInForce  domain.TimeInForce
	BaseQuantity float64
	Price        float64
}

type StopDetails struct {
	Type        domain.OrderType
	TimeInForce domain.TimeInForce
	QuantityPct float64
	Price       float64
}

type CancelOrderRequest struct {
	Pair         domain.Pair
	OrderID      string
	TakeProfitID string
	StopLossID   string
}

type ExchangeInfoer[T any] interface {
	Exists(name string) bool
	Save(name string, data T) error
	Read(name string) (T, error)
}
