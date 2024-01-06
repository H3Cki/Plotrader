package binancefutures

import (
	"strconv"
	"time"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/go-binance/v2/futures"
)

type exchangeOrder struct {
	O futures.Order `json:"order"`
}

func newEo(order futures.Order) exchangeOrder {
	return exchangeOrder{O: order}
}

func (eo exchangeOrder) Status() domain.ExchangeOrderStatus {
	switch eo.O.Status {
	case futures.OrderStatusTypeNew:
		return domain.ExchangeOrderStatusOpen
	case futures.OrderStatusTypePartiallyFilled:
		return domain.ExchangeOrderStatusPartiallyFilled
	case futures.OrderStatusTypeFilled:
		return domain.ExchangeOrderStatusFilled
	case futures.OrderStatusTypeCanceled,
		futures.OrderStatusTypeRejected,
		futures.OrderStatusTypeExpired:
		return domain.ExchangeOrerStatusCanceled
	}
	return ""
}

func (eo exchangeOrder) ID() string {
	return eo.O.ClientOrderID
}

func (eo exchangeOrder) CreatedAt() time.Time {
	return time.Unix(eo.O.Time, 0)
}

func (eo exchangeOrder) StopPrice() float64 {
	fPrice, err := strconv.ParseFloat(eo.O.StopPrice, 64)
	if err != nil {
		///
	}
	return fPrice
}

func (eo exchangeOrder) Price() float64 {
	fPrice, err := strconv.ParseFloat(eo.O.Price, 64)
	if err != nil {
		///
	}
	return fPrice
}

func (eo exchangeOrder) BaseQuantity() float64 {
	fQty, err := strconv.ParseFloat(eo.O.OrigQuantity, 64)
	if err != nil {
		///
	}
	return fQty
}
