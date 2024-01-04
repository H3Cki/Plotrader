package binancefutures

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/H3Cki/go-binance/v2/futures"
)

// Parse functions

func pairToSymbol(p domain.Pair) string {
	return p.Base + p.Quote
}

func parseSide(s domain.PositionSide) (futures.SideType, error) {
	switch s {
	case domain.PositionSideLong:
		return futures.SideTypeBuy, nil
	case domain.PositionSideShort:
		return futures.SideTypeSell, nil
	}
	return "", fmt.Errorf("unexpected order side: %s", s)
}

func orderSide(side domain.PositionSide, opposite bool) (futures.SideType, error) {
	switch side {
	case domain.PositionSideLong:
		if opposite {
			return futures.SideTypeSell, nil
		}
		return futures.SideTypeBuy, nil
	case domain.PositionSideShort:
		if opposite {
			return futures.SideTypeBuy, nil
		}
		return futures.SideTypeSell, nil
	}
	return "", fmt.Errorf("unsupported position side: %s", side)
}

// Conversion functions

func corToOrderService(c *futures.Client, req outbound.CreateOrdersRequest, symbol futures.Symbol) (*futures.CreateOrderService, error) {
	orderSide, err := parseSide(req.Side)
	if err != nil {
		return nil, err
	}

	cro := createOrderRequest{
		symbol:       symbol,
		side:         orderSide,
		orderType:    futures.OrderTypeLimit,
		price:        req.Order.Price,
		baseQuantity: req.Order.BaseQuantity,
		timeInForce:  futures.TimeInForceTypeGTC,
	}

	if err := applyFilters(&cro); err != nil {
		return nil, err
	}

	svc := c.NewCreateOrderService().Symbol(symbol.Symbol).
		Side(cro.side).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(fmt.Sprint(cro.baseQuantity)).
		Price(fmt.Sprint(cro.price))

	return svc, nil
}

func corToTakeProfitServices(c *futures.Client, req outbound.CreateOrdersRequest, symbol futures.Symbol) ([]*futures.CreateOrderService, error) {
	side, err := orderSide(req.Side, true)
	if err != nil {
		return nil, err
	}

	svcs := []*futures.CreateOrderService{}
	for _, r := range req.TakeProfit {
		cro := createOrderRequest{
			symbol:       symbol,
			side:         side,
			orderType:    futures.OrderTypeTakeProfitMarket,
			price:        r.StopPrice,
			baseQuantity: r.BaseQuantity,
			timeInForce:  futures.TimeInForceTypeGTC,
		}

		if err := applyFilters(&cro); err != nil {
			return nil, err
		}

		svc := c.NewCreateOrderService().
			Symbol(symbol.Symbol).
			Side(cro.side).
			Type(futures.OrderTypeTakeProfitMarket).
			TimeInForce(futures.TimeInForceTypeGTC).
			Quantity(fmt.Sprint(cro.baseQuantity)).
			StopPrice(fmt.Sprint(cro.price))

		svcs = append(svcs, svc)
	}

	return svcs, nil
}

func corToStopLossServices(c *futures.Client, req outbound.CreateOrdersRequest, symbol futures.Symbol) ([]*futures.CreateOrderService, error) {
	side, err := orderSide(req.Side, true)
	if err != nil {
		return nil, err
	}

	svcs := []*futures.CreateOrderService{}
	for _, r := range req.StopLoss {
		cro := createOrderRequest{
			symbol:       symbol,
			side:         side,
			orderType:    futures.OrderTypeStopMarket,
			price:        r.StopPrice,
			baseQuantity: r.BaseQuantity,
			timeInForce:  futures.TimeInForceTypeGTC,
		}

		if err := applyFilters(&cro); err != nil {
			return nil, err
		}

		svc := c.NewCreateOrderService().
			Symbol(symbol.Symbol).
			Side(cro.side).
			Type(futures.OrderTypeStopMarket).
			TimeInForce(futures.TimeInForceTypeGTC).
			Quantity(fmt.Sprint(cro.baseQuantity)).
			StopPrice(fmt.Sprint(cro.price))

		svcs = append(svcs, svc)
	}

	return svcs, nil
}

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

func modRespAt(resp *futures.ModifyMultipleOrdersResponse, idx int) (*futures.Order, error) {
	if idx >= resp.N {
		return nil, errors.New("index out of range")
	}
	return resp.Orders[idx], resp.Errors[idx]
}

func toOrderModification(mod outbound.OrderModification) futures.OrderModification {
	o := mod.ExchangeOrder.(*exchangeOrder)
	return futures.OrderModification{
		OrderID:      &o.order.OrderID,
		Symbol:       &o.order.Symbol,
		Side:         (*string)(&o.order.Side),
		BaseQuantity: &mod.BaseQuantity,
		Price:        &mod.Price,
	}
}
