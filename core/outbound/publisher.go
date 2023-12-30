package outbound

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
)

type Publisher interface {
	PublishOrderUpdate(context.Context, OrderUpdate) error
}

type OrderUpdate struct {
	domain.Order
}

type OrderRequest struct {
	Type        domain.OrderType   `json:"type" validate:"required"`
	TimeInForce domain.TimeInForce `json:"timeInForce" validate:"required"`
	Plot        map[string]any     `json:"plot" validate:"required"`
}
