package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/domain/geometry"
)

type FollowService interface {
	CreateFollow(context.Context, CreateFollowRequest) (CreateFollowResponse, error)
	GetFollow(context.Context, GetFollowRequest) (GetFollowResponse, error)
	StopFollow(context.Context, StopFollowRequest) error
}

type CreateFollowRequest struct {
	Exchange Exchange             `json:"exchange" validate:"required"`
	Symbol   string               `json:"symbol" validate:"required"`
	Interval string               `json:"interval" validate:"required"`
	Orders   []CreateOrderRequest `json:"orders" validate:"required"`

	WebhookURL string `json:"webhookURL"`
}

type CreateOrderRequest struct {
	Name          string                  `json:"name"`
	Type          domain.OrderType        `json:"type" validate:"required"`
	Side          domain.OrderSide        `json:"side" validate:"required"`
	QuoteQuantity float64                 `json:"quoteQuantity"`
	BaseQuantity  float64                 `json:"baseQuantity"`
	ClosePosition bool                    `json:"closePosition"`
	ReduceOnly    bool                    `json:"reduceOnly"`
	Relations     []domain.StatusRelation `json:"relations"`
	PlotSpec      geometry.PlotSpec       `json:"plot" validate:"required"`
}

type Exchange struct {
	Name      string         `json:"name" validate:"required"`
	ConfigEnv string         `json:"configEnv"`
	Config    map[string]any `json:"config"`
}

func (e *Exchange) UnmarshalConfig(to any) error {
	var cfgBytes []byte

	if e.ConfigEnv != "" {
		v, ok := os.LookupEnv(e.ConfigEnv)
		if !ok {
			return fmt.Errorf("ENV %s not set", e.ConfigEnv)
		}

		cb, err := json.Marshal(v)
		if err != nil {
			return err
		}
		cfgBytes = cb
	} else {
		bytes, err := json.Marshal(e.Config)
		if err != nil {
			return err
		}
		cfgBytes = bytes
	}

	return json.Unmarshal(cfgBytes, to)
}

type CreateFollowResponse struct {
	FollowID string `json:"followID"`
}

type GetFollowRequest struct {
	Exchange Exchange `json:"exchange" validate:"required"`
	FollowID string   `json:"followID" validate:"required"`
}

type GetFollowResponse struct {
	Follow domain.Follow `json:"follow"`
}

type StopFollowRequest struct {
	Exchange     Exchange `json:"exchange" validate:"required"`
	FollowID     string   `json:"followID" validate:"required"`
	CancelOrders bool     `json:"cancelOrders"`
}
