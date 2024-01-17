package outbound

import (
	"context"

	"github.com/H3Cki/Plotrader/core/domain"
)

type Publisher interface {
	PublishFollowUpdate(context.Context, FollowUpdate) error
}

type FollowUpdate struct {
	domain.Follow
}
