package outbound

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
)

type Publisher interface {
	PublishOrderUpdate(context.Context, OrderUpdate) error
}

type OrderUpdate struct {
	domain.Follow
}
