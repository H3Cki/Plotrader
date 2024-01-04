package outbound

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
)

type Exchange interface {
	Init(context.Context) error
	GetPrice(context.Context, GetPriceRequest) (float64, error)
	CreateOrders(context.Context, CreateOrdersRequest) (ExchangeOrders, error)
	ModifyOrders(context.Context, ModifyOrdersRequest) (ExchangeOrders, error)
	CancelOrders(context.Context, CancelOrdersRequest) error
}

type GetPriceRequest struct {
	Pair domain.Pair
}

type CreateOrdersRequest struct {
	Pair       domain.Pair
	Side       domain.PositionSide
	Order      OrderDetails
	TakeProfit []OrderDetails
	StopLoss   []OrderDetails
}

type OrderDetails struct {
	BaseQuantity float64
	Price        float64
	StopPrice    float64
}

type ModifyOrdersRequest struct {
	Order      OrderModification
	TakeProfit []OrderModification
	StopLoss   []OrderModification
}

type OrderModification struct {
	ExchangeOrder domain.ExchangeOrder
	OrderDetails
}

type CancelOrdersRequest struct {
	ExchangeOrder      domain.ExchangeOrder
	TPSLExchangeOrders []domain.ExchangeOrder
}

type ExchangeInfoer[T any] interface {
	Exists(name string) bool
	Save(name string, data T) error
	Read(name string) (T, error)
}

type ExchangeOrders struct {
	Order       domain.ExchangeOrder
	TakeProfits []domain.ExchangeOrder
	StopLosses  []domain.ExchangeOrder
}
