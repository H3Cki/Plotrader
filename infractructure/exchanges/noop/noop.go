package noop

import (
	"context"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
)

type Exchange struct {
	Price float64 `json:"price"`
}

func (e *Exchange) Init(_ context.Context) error {
	return nil
}

func (e *Exchange) GetPrice(_ context.Context, _ outbound.GetPriceRequest) (float64, error) {
	return e.Price, nil
}

func (e *Exchange) CreateOrder(_ context.Context, _ outbound.CreateOrderRequest) (domain.ExchangeOrders, error) {
	return domain.ExchangeOrders{
		Time:       time.Time{},
		Order:      domain.ExchangeOrder{ID: "noop"},
		TakeProfit: domain.ExchangeOrder{},
		StopLoss:   domain.ExchangeOrder{},
	}, nil
}

func (e *Exchange) CancelOrder(_ context.Context, _ outbound.CancelOrderRequest) error {
	return nil
}
