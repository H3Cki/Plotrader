package binancefutures

import (
	"encoding/json"
	"fmt"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/go-binance/v2/futures"
)

// Parse functions

func pairToSymbol(p domain.Pair) string {
	return p.Base + p.Quote
}

func oppositeSide(s futures.SideType) futures.SideType {
	switch s {
	case futures.SideTypeBuy:
		return futures.SideTypeSell
	case futures.SideTypeSell:
		return futures.SideTypeBuy
	}
	return ""
}

func orderSide(side domain.Side, opposite bool) (futures.SideType, error) {
	switch side {
	case domain.SideLong:
		s := futures.SideTypeBuy
		if opposite {
			s = oppositeSide(s)
		}
		return s, nil
	case domain.SideShort:
		s := futures.SideTypeSell
		if opposite {
			s = oppositeSide(s)
		}
		return s, nil
	}
	return "", fmt.Errorf("unsupported position side: %s", side)
}

// Conversion functions
func parseOrderIdentification(data any) (orderIdentification, error) {
	orderId := orderIdentification{}
	orderDataBytes, err := json.Marshal(data)
	if err != nil {
		return orderIdentification{}, err
	}
	if err := json.Unmarshal(orderDataBytes, &orderId); err != nil {
		return orderIdentification{}, err
	}
	return orderId, nil
}

func corToOrder(cor *futures.CreateOrderResponse) *futures.Order {
	return &futures.Order{
		Symbol:           cor.Symbol,
		OrderID:          cor.OrderID,
		ClientOrderID:    cor.ClientOrderID,
		Price:            cor.Price,
		ReduceOnly:       cor.ReduceOnly,
		OrigQuantity:     cor.OrigQuantity,
		ExecutedQuantity: cor.ExecutedQuantity,
		CumQuantity:      "0", //??
		CumQuote:         cor.CumQuote,
		Status:           cor.Status,
		TimeInForce:      cor.TimeInForce,
		Type:             cor.Type,
		Side:             cor.Side,
		StopPrice:        cor.StopPrice,
		Time:             cor.UpdateTime,
		UpdateTime:       cor.UpdateTime,
		WorkingType:      cor.WorkingType,
		ActivatePrice:    cor.ActivatePrice,
		PriceRate:        cor.PriceRate,
		AvgPrice:         cor.AvgPrice,
		OrigType:         string(cor.Type),
		PositionSide:     cor.PositionSide,
		PriceProtect:     cor.PriceProtect,
		ClosePosition:    cor.ClosePosition,
	}
}

func morToOrder(cor *futures.ModifyOrderResponse) *futures.Order {
	return &futures.Order{
		Symbol:           cor.Symbol,
		OrderID:          cor.OrderID,
		ClientOrderID:    cor.ClientOrderID,
		Price:            cor.Price,
		ReduceOnly:       cor.ReduceOnly,
		OrigQuantity:     cor.OrigQuantity,
		ExecutedQuantity: cor.ExecutedQuantity,
		CumQuantity:      "0", //??
		CumQuote:         cor.CumQuote,
		Status:           cor.Status,
		TimeInForce:      cor.TimeInForce,
		Type:             cor.Type,
		OrigType:         string(cor.OrigType),
		Side:             cor.Side,
		StopPrice:        cor.StopPrice,
		Time:             cor.UpdateTime,
		UpdateTime:       cor.UpdateTime,
		WorkingType:      cor.WorkingType,
		ActivatePrice:    cor.ActivatePrice,
		PriceRate:        cor.PriceRate,
		AvgPrice:         cor.AvgPrice,
		PositionSide:     cor.PositionSide,
		PriceProtect:     cor.PriceProtect,
		ClosePosition:    cor.ClosePosition,
	}
}
