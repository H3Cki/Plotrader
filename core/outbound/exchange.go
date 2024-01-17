package outbound

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
)

type Exchange interface {
	Init(context.Context) error
	GetOrder(context.Context, GetExchangeOrderRequest) (*domain.ExchangeOrder, error)
	CreateOrder(context.Context, CreateExchangeOrderRequest) (*domain.ExchangeOrder, error)
	ModifyOrder(context.Context, ModifyExchangeOrderRequest) (*domain.ExchangeOrder, error)
	CancelOrder(context.Context, CancelExchangeOrdersRequest) (*domain.ExchangeOrder, error)
}

type GetExchangeOrderRequest struct {
	EO domain.ExchangeOrder
}

type CreateExchangeOrderRequest struct {
	Pair         domain.Pair
	Type         domain.OrderType
	Side         domain.OrderSide
	BaseQuantity float64
	Price        float64
	StopPrice    float64
}

type ModifyExchangeOrderRequest struct {
	EO           *domain.ExchangeOrder // EO contains data that lets the exchange identify the order
	BaseQuantity float64
	Price        float64
	StopPrice    float64
}

type CancelExchangeOrdersRequest struct {
	EO *domain.ExchangeOrder
}

type FileLoader[T any] interface {
	Exists(name string) bool
	Save(name string, data T) error
	Read(name string) (T, error)
}
