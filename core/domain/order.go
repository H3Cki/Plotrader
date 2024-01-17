package domain

import (
	"github.com/H3Cki/Plotrader/core/domain/geometry"
)

type OrderType string

var (
	OrderTypeLimit      OrderType = "LIMIT"
	OrderTypeMarket     OrderType = "MARKET"
	OrderTypeTakeProfit OrderType = "TAKE_PROFIT"
	OrderTypeStopLoss   OrderType = "STOP_LOSS"
)

type OrderSide string

var (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

type OrderStatus string

var (
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusActive     OrderStatus = "ACTIVE"
	OrderStatusDone       OrderStatus = "DONE"
	OrderStatusCanceled   OrderStatus = "CANCELED"
)

type Order struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Pair          Pair              `json:"pair"`
	Status        OrderStatus       `json:"status"`
	Type          OrderType         `json:"type"`
	Side          OrderSide         `json:"side"`
	QuoteQuantity float64           `json:"quoteQuantity"`
	BaseQuantity  float64           `json:"baseQuantity"`
	ClosePosition bool              `json:"closePosition"`
	ReduceOnly    bool              `json:"reduceOnly"`
	Relations     []StatusRelation  `json:"relations"`
	PlotSpec      geometry.PlotSpec `json:"plotSpec"`
	Plot          geometry.Plot     `json:"-" bson:"-"`

	ExchangeHash  string         `json:"exchangeHash"`
	ExchangeOrder *ExchangeOrder `json:"exchangeOrder"`
}

type RelationCondition string

var (
	RelationConditionEqual    RelationCondition = "EQUAL"
	RelationConditionNotEqual RelationCondition = "NOT_EQUAL"
)

type StatusRelation struct {
	OrderName string            `json:"orderName"`
	Status    OrderStatus       `json:"status"`
	Condition RelationCondition `json:"condition"`
}
