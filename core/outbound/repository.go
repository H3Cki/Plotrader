package outbound

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
)

type Repository interface {
	Connect(context.Context) error
	Disconnect(context.Context) error
	// Follow
	CreateFollow(context.Context, CreateFollowRequest) error
	GetFollow(context.Context, GetFollowRequest) (domain.Follow, error)
	UpdateFollow(context.Context, UpdateFollowRequest) error

	// Order
	CreateOrder(context.Context, CreateOrderRequest) error
	GetOrder(context.Context, GetOrderRequest) (domain.Order, error)
	UpdateOrder(context.Context, UpdateOrderRequest) error
}

type CreateFollowRequest struct {
	Follow domain.Follow
}

type CreateOrderRequest struct {
	Order domain.Order
}

type GetOrderRequest struct {
	OrderID string
}

type UpdateOrderRequest struct {
	Order domain.Order
}

type GetFollowRequest struct {
	FollowID string
}

type UpdateFollowRequest struct {
	Follow domain.Follow
}
