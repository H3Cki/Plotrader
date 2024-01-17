package followsvc

import (
	"slices"
	"strings"

	"github.com/H3Cki/Plotrader/core/domain"
	"github.com/pkg/errors"
)

func orderSideForType(baseSide domain.OrderSide, orderType domain.OrderType) domain.OrderSide {
	switch orderType {
	case domain.OrderTypeLimit, domain.OrderTypeMarket:
		return baseSide
	case domain.OrderTypeTakeProfit, domain.OrderTypeStopLoss:
		return oppositeSide(baseSide)
	}
	panic(baseSide)
}

func oppositeSide(s domain.OrderSide) domain.OrderSide {
	if s == domain.OrderSideBuy {
		return domain.OrderSideSell
	}
	return domain.OrderSideBuy
}

func parsePair(s string) (domain.Pair, error) {
	bq := strings.Split(s, "-")
	if len(bq) != 2 {
		return domain.Pair{}, errors.New("invalid symbol")
	}

	return domain.Pair{
		Base:  bq[0],
		Quote: bq[1],
	}, nil
}

func baseQuantity(price, base, quote float64) float64 {
	if base != 0 {
		return base
	}

	return quote / price
}

func replaceOrders(orig []domain.Order, new []domain.Order) []domain.Order {
	for _, n := range new {
		idx := slices.IndexFunc(orig, func(o domain.Order) bool {
			return o.ID == n.ID
		})
		if idx == -1 {
			continue
		}
		orig[idx] = n
	}
	return orig
}
