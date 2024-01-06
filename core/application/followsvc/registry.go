package followsvc

import (
	"slices"
	"sync"

	"github.com/H3Cki/Plotrader/core/domain"
)

type registry struct {
	followMap map[string]*modder
	mu        *sync.Mutex
}

func newRegistry() *registry {
	return &registry{
		followMap: map[string]*modder{},
		mu:        &sync.Mutex{},
	}
}

func (r *registry) setFollow(f domain.Follow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.followMap[f.ID] = newModder(f)
}

func (r *registry) getFollow(id string) domain.Follow {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.followMap[id].getFollow()
}

func (r *registry) getModder(id string) *modder {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.followMap[id]
}

type modder struct {
	follow  *domain.Follow
	stopMap map[string]domain.StopOrder
	mu      *sync.Mutex
}

func newModder(f domain.Follow) *modder {
	m := &modder{
		follow: &f,
		mu:     &sync.Mutex{},
	}
	m.buildStopMap()
	return m
}

func (m *modder) buildStopMap() {
	stopMap := map[string]domain.StopOrder{}
	for _, stop := range m.follow.TakeProfits {
		stopMap[stop.ID] = stop
	}
	for _, stop := range m.follow.StopLosses {
		stopMap[stop.ID] = stop
	}
	m.stopMap = stopMap
}

func (m *modder) getParentOrder() domain.ParentOrder {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.follow.Order
}

func (m *modder) setParentOrder(o domain.ParentOrder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.follow.Order = o
}

func (m *modder) setStop(stop domain.StopOrder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopMap[stop.ID] = stop
}

func (m *modder) getStop(id string) domain.StopOrder {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopMap[id]
}

func (m *modder) getFollow() domain.Follow {
	m.mu.Lock()
	defer m.mu.Unlock()
	tps := slices.Clone(m.follow.TakeProfits)
	for i, tp := range tps {
		tps[i] = m.stopMap[tp.ID]
	}
	sls := slices.Clone(m.follow.StopLosses)
	for i, sl := range sls {
		sls[i] = m.stopMap[sl.ID]
	}
	return domain.Follow{
		ID:           m.follow.ID,
		Exchange:     m.follow.Exchange,
		Pair:         m.follow.Pair,
		PositionSide: m.follow.PositionSide,
		Interval:     m.follow.Interval,
		LastTick:     m.follow.LastTick,
		WebhookURL:   m.follow.WebhookURL,
		Order:        m.follow.Order,
		TakeProfits:  tps,
		StopLosses:   sls,
	}
}

func (m *modder) setFollow(f domain.Follow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.follow = &f
	m.buildStopMap()
}
