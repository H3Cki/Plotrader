package binancefutures

import (
	"fmt"
	"strconv"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/go-binance/v2/futures"
)

// Parse functions

func pairToSymbol(p domain.Pair) string {
	return p.Base + p.Quote
}

func orderSide(side domain.OrderSide) (futures.SideType, error) {
	switch side {
	case domain.OrderSideBuy:
		return futures.SideTypeBuy, nil

	case domain.OrderSideSell:
		return futures.SideTypeSell, nil
	}
	return "", fmt.Errorf("unsupported position side: %s", side)
}

func orderType(side domain.OrderType) (futures.OrderType, error) {
	switch side {
	case domain.OrderTypeLimit:
		return futures.OrderTypeLimit, nil
	case domain.OrderTypeStopLoss:
		return futures.OrderTypeStopMarket, nil
	case domain.OrderTypeTakeProfit:
		return futures.OrderTypeTakeProfitMarket, nil
	}
	return "", fmt.Errorf("unsupported position side: %s", side)
}

func orderToOrder(order *futures.Order) (*domain.ExchangeOrder, error) {
	status, price, quantity, err := statusPriceQuantity(order.Status, order.Price, order.OrigQuantity)
	if err != nil {
		return nil, err
	}

	return &domain.ExchangeOrder{
		ID:           order.OrderID,
		Status:       status,
		Type:         string(order.Type),
		Side:         string(order.Side),
		Symbol:       order.Symbol,
		Price:        price,
		BaseQuantity: quantity,
	}, nil
}

func createRespToOrder(resp *futures.CreateOrderResponse) (*domain.ExchangeOrder, error) {
	status, price, quantity, err := statusPriceQuantity(resp.Status, resp.Price, resp.OrigQuantity)
	if err != nil {
		return nil, err
	}

	return &domain.ExchangeOrder{
		ID:           resp.OrderID,
		Status:       status,
		Type:         string(resp.Type),
		Side:         string(resp.Side),
		Symbol:       resp.Symbol,
		Price:        price,
		BaseQuantity: quantity,
	}, nil
}

func cancelRespToOrder(resp *futures.CancelOrderResponse) (*domain.ExchangeOrder, error) {
	status, price, quantity, err := statusPriceQuantity(resp.Status, resp.Price, resp.OrigQuantity)
	if err != nil {
		return nil, err
	}

	return &domain.ExchangeOrder{
		ID:           resp.OrderID,
		Status:       status,
		Type:         string(resp.Type),
		Side:         string(resp.Side),
		Symbol:       resp.Symbol,
		Price:        price,
		BaseQuantity: quantity,
	}, nil
}

func modifyRespToOrder(resp *futures.ModifyOrderResponse) (*domain.ExchangeOrder, error) {
	status, price, quantity, err := statusPriceQuantity(resp.Status, resp.Price, resp.OrigQuantity)
	if err != nil {
		return nil, err
	}

	return &domain.ExchangeOrder{
		ID:           resp.OrderID,
		Status:       status,
		Type:         string(resp.Type),
		Side:         string(resp.Side),
		Symbol:       resp.Symbol,
		Price:        price,
		BaseQuantity: quantity,
	}, nil
}

func statusPriceQuantity(status futures.OrderStatusType, price, quantity string) (domain.OrderStatus, float64, float64, error) {
	s, err := toDomainOrderStatus(status)
	if err != nil {
		return "", 0, 0, err
	}

	p, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return "", 0, 0, err
	}

	q, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		return "", 0, 0, err
	}

	return s, p, q, nil
}

func toDomainOrderStatus(status futures.OrderStatusType) (domain.OrderStatus, error) {
	switch status {
	case futures.OrderStatusTypeNew, futures.OrderStatusTypePartiallyFilled:
		return domain.OrderStatusActive, nil
	case futures.OrderStatusTypeFilled:
		return domain.OrderStatusDone, nil
	case futures.OrderStatusTypeCanceled,
		futures.OrderStatusTypeRejected,
		futures.OrderStatusTypeExpired:
		return domain.OrderStatusCanceled, nil
	}
	return "", fmt.Errorf("unknown order status: %s", status)
}
