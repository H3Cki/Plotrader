package outbound

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
)

type ExchangeInfoer[T any] interface {
	Exists(name string) bool
	Save(name string, data T) error
	Read(name string) (T, error)
}

type Exchange interface {
	Init(context.Context) error
	GetOrder(context.Context, domain.ExchangeOrder) (domain.ExchangeOrder, error)

	// Main Order
	CreateOrder(context.Context, CreateOrderRequest) (domain.ExchangeOrder, error)
	ModifyOrder(context.Context, ModifyOrderRequest) (domain.ExchangeOrder, error)

	// TakeProfit
	CreateTakeProfitOrder(context.Context, CreateTakeProfitRequest) (domain.ExchangeOrder, error)
	ModifyTakeProfitOrder(context.Context, ModifyTakeProfitRequest) (domain.ExchangeOrder, error)

	// StopLoss
	CreateStopLossOrder(context.Context, CreateStopLossRequest) (domain.ExchangeOrder, error)
	ModifyStopLossOrder(context.Context, ModifyStopLossRequest) (domain.ExchangeOrder, error)

	// Batch
	CancelOrder(context.Context, domain.ExchangeOrder) error
}

type CreateStopOrderRequest struct {
	Request StopRequest
}

type StopRequest struct {
	Type         domain.StopType
	Parent       domain.ExchangeOrder
	BaseQuantity float64
	StopPrice    float64
	//Price        float64
}

type CreateOrderRequest struct {
	Pair    domain.Pair
	PosSide domain.PositionSide
	Request OrderRequest
}

type ModifyOrderRequest struct {
	ExchangeOrder domain.ExchangeOrder
	Request       OrderRequest
}

type OrderRequest struct {
	BaseQuantity float64
	Price        float64
}

type CreateTakeProfitRequest struct {
	Parent  domain.ExchangeOrder
	Request TakeProfitRequest
}

type ModifyTakeProfitRequest struct {
	Parent        domain.ExchangeOrder
	ExchangeOrder domain.ExchangeOrder
	Request       TakeProfitRequest
}

type TakeProfitRequest struct {
	BaseQuantity float64
	Price        float64
	StopPrice    float64
}

type CreateStopLossRequest struct {
	Parent  domain.ExchangeOrder
	Request StopLossRequest
}

type ModifyStopLossRequest struct {
	Parent        domain.ExchangeOrder
	ExchangeOrder domain.ExchangeOrder
	Request       StopLossRequest
}

type StopLossRequest struct {
	BaseQuantity float64
	Price        float64
	StopPrice    float64
}

type CancelOrdersRequest struct {
	ExchangeOrders []domain.ExchangeOrder
}

type CancelOrderRequest struct {
	ExchangeOrder domain.ExchangeOrder
}
