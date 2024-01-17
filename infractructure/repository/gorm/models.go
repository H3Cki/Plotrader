package gormrepo

import (
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/domain/geometry"
	"gorm.io/gorm"
)

type Pair struct {
	gorm.Model
	Base  string
	Quote string
}

type Follow struct {
	ID           string `gorm:"primarykey"`
	Status       domain.FollowStatus
	ExchangeHash string
	Pair         Pair
	Interval     time.Duration
	WebhookURL   string
	OrderIDs     []string
}

func followFromDomain(follow domain.Follow) *Follow {
	return &Follow{
		ID:           follow.ID,
		Status:       follow.Status,
		ExchangeHash: follow.ExchangeHash,
		Pair:         Pair{Base: follow.Pair.Base, Quote: follow.Pair.Quote},
		Interval:     follow.Interval,
		WebhookURL:   follow.WebhookURL,
		OrderIDs:     follow.OrderIDs,
	}
}

func (f *Follow) domain() domain.Follow {
	return domain.Follow{
		ID:           f.ID,
		Status:       f.Status,
		ExchangeHash: f.ExchangeHash,
		Pair:         domain.Pair{Base: f.Pair.Base, Quote: f.Pair.Quote},
		Interval:     f.Interval,
		WebhookURL:   f.WebhookURL,
		OrderIDs:     f.OrderIDs,
	}
}

type FollowOrderJoiner struct {
	FollowID string `gorm:"foreignkey"`
}

type Order struct {
	ID            string `gorm:"primarykey"`
	FollowID      string `gorm:"foreignkey"`
	Name          string `gorm:"index"`
	Pair          Pair
	Status        domain.OrderStatus
	Type          domain.OrderType
	Side          domain.OrderSide
	QuoteQuantity float64
	BaseQuantity  float64
	ClosePosition bool
	ReduceOnly    bool
	Relations     []domain.StatusRelation
	Plot          geometry.Plot

	ExchangeHash  string
	ExchangeOrder *domain.ExchangeOrder
}

func orderFromDomain(order domain.Order) *Order {
	return &Order{
		ID:            order.ID,
		Name:          order.Name,
		Pair:          Pair{Base: order.Pair.Base, Quote: order.Pair.Quote},
		Status:        order.Status,
		Type:          order.Type,
		Side:          order.Side,
		QuoteQuantity: order.BaseQuantity,
		BaseQuantity:  order.QuoteQuantity,
		ClosePosition: order.ClosePosition,
		ReduceOnly:    order.ReduceOnly,
		Relations:     order.Relations,
		Plot:          order.Plot,
		ExchangeHash:  order.ExchangeHash,
		ExchangeOrder: order.ExchangeOrder,
	}
}

func (o *Order) domain() domain.Order {
	return domain.Order{
		ID:            o.ID,
		Name:          o.Name,
		Pair:          domain.Pair{Base: o.Pair.Base, Quote: o.Pair.Quote},
		Status:        o.Status,
		Type:          o.Type,
		Side:          o.Side,
		QuoteQuantity: o.QuoteQuantity,
		BaseQuantity:  o.BaseQuantity,
		ClosePosition: o.ClosePosition,
		ReduceOnly:    o.ReduceOnly,
		Relations:     o.Relations,
		Plot:          o.Plot,
		ExchangeHash:  o.ExchangeHash,
		ExchangeOrder: o.ExchangeOrder,
	}
}
