package followsvc

import (
	"sync"

	"github.com/H3Cki/Plotrader/core/domain"
)

type parentMod func(f domain.ParentOrder) domain.ParentOrder
type stopMod func(f domain.StopOrder) domain.StopOrder

type followManager struct {
	f  *domain.Follow
	mu *sync.Mutex
	*sync.Mutex
}

func newManager(f *domain.Follow) *followManager {
	return &followManager{
		f:     f,
		mu:    &sync.Mutex{},
		Mutex: &sync.Mutex{},
	}
}

func (m *followManager) follow() domain.Follow {
	m.mu.Lock()
	defer m.mu.Unlock()
	return *m.f
}

func (m *followManager) getOrder() domain.ParentOrder {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.f.Order
}

func (m *followManager) getTPs() []domain.StopOrder {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.f.TakeProfits
}

func (m *followManager) updateTP(idx int, mods ...stopMod) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tp := m.f.TakeProfits[idx]
	for _, mod := range mods {
		tp = mod(tp)
	}
	m.f.TakeProfits[idx] = tp
}

func (m *followManager) getSLs() []domain.StopOrder {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.f.StopLosses
}

func (m *followManager) updateSL(idx int, mods ...stopMod) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sl := m.f.StopLosses[idx]
	for _, mod := range mods {
		sl = mod(sl)
	}
	m.f.StopLosses[idx] = sl
}

func (m *followManager) updateOrder(mods ...parentMod) {
	m.mu.Lock()
	defer m.mu.Unlock()
	order := m.f.Order
	for _, mod := range mods {
		order = mod(order)
	}
	m.f.Order = order
}

func withOrderStatus(status domain.OrderStatus) parentMod {
	return func(o domain.ParentOrder) domain.ParentOrder {
		o.Status = status
		return o
	}
}

func withOrderEo(eo domain.ExchangeOrder) parentMod {
	return func(o domain.ParentOrder) domain.ParentOrder {
		o.ExchangeOrder = eo
		return o
	}
}

func withStopStatus(status domain.StopStatus) stopMod {
	return func(o domain.StopOrder) domain.StopOrder {
		o.Status = status
		return o
	}
}

func withStopEo(eo domain.ExchangeOrder) stopMod {
	return func(o domain.StopOrder) domain.StopOrder {
		o.ExchangeOrder = eo
		return o
	}
}
