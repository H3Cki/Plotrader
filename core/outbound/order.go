package outbound

import (
	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/domain/geometry"
)

type Order struct {
	domain.Order
	Plot geometry.PlotSpec `json:"plot"`
}
