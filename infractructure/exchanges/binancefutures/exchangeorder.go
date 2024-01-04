package binancefutures

import (
	"encoding/json"
	"strconv"

	"github.com/H3Cki/go-binance/v2/futures"
)

type exchangeOrder struct {
	order *futures.Order
	err   error
}

func newEo(order *futures.Order, err error) *exchangeOrder {
	return &exchangeOrder{order: order, err: err}
}

func (eo *exchangeOrder) StopPrice() float64 {
	fPrice, err := strconv.ParseFloat(eo.order.StopPrice, 64)
	if err != nil {
		///
	}
	return fPrice
}

func (eo *exchangeOrder) Price() float64 {
	fPrice, err := strconv.ParseFloat(eo.order.Price, 64)
	if err != nil {
		///
	}
	return fPrice
}

func (eo *exchangeOrder) BaseQuantity() float64 {
	fQty, err := strconv.ParseFloat(eo.order.OrigQuantity, 64)
	if err != nil {
		///
	}
	return fQty
}

func (eo *exchangeOrder) Error() error {
	return eo.err
}

func parseExchangeOrder(data any) (exchangeOrder, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return exchangeOrder{}, err
	}
	eo := exchangeOrder{}
	if err := json.Unmarshal(bytes, &eo); err != nil {
		return exchangeOrder{}, err
	}
	return eo, nil
}
