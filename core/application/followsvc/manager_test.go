package followsvc

import (
	"sync"
	"testing"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/stretchr/testify/assert"
)

var o1 = domain.ParentOrder{
	ID:            "1",
	Status:        "2",
	ExchangeOrder: nil,
	QuoteQuantity: 3,
	BaseQuantity:  4,
	Plot:          nil,
}

func Test_followManager_updateOrder(t *testing.T) {

	tests := []struct {
		name     string
		order    domain.ParentOrder
		mods     []parentMod
		expected domain.ParentOrder
	}{
		{
			name:     "1",
			order:    order(),
			mods:     []parentMod{},
			expected: o1,
		},
		{
			name:  "1",
			order: order(),
			mods:  []parentMod{withOrderStatus(domain.OrderStatusPending)},
			expected: domain.ParentOrder{
				ID:            "1",
				Status:        domain.OrderStatusPending,
				ExchangeOrder: nil,
				QuoteQuantity: 3,
				BaseQuantity:  4,
				Plot:          nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &followManager{
				f:  &domain.Follow{Order: tt.order},
				mu: &sync.Mutex{},
			}

			m.updateOrder(tt.mods...)
			after := m.f.Order
			assert.EqualValues(t, tt.expected, after)
		})
	}
}

func order() domain.ParentOrder {
	return o1
}
