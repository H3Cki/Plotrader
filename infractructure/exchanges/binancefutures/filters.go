package binancefutures

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/H3Cki/go-binance/v2/futures"
)

var orderTypeFilters = map[futures.OrderType]func(*orderValues) error{
	futures.OrderTypeLimit: func(req *orderValues) error {
		s := req.symbol
		// PRICE
		if pf := s.PriceFilter(); pf != nil {
			price, err := priceFilter(pf, req.price)
			if err != nil {
				return err
			}

			req.price = price
		}

		// LOT SIZE
		if lsf := s.LotSizeFilter(); lsf != nil {
			qty, err := lotSizeFilter(lsf, req.baseQuantity)
			if err != nil {
				return err
			}

			req.baseQuantity = qty
		}

		// MIN NOTIONAL
		if mnf := s.MinNotionalFilter(); mnf != nil {
			err := minNotionalFilter(mnf, req.price, req.baseQuantity)
			if err != nil {
				return err
			}
		}

		return nil
	},
	futures.OrderTypeStopMarket:       marketOrderFilters,
	futures.OrderTypeTakeProfitMarket: marketOrderFilters,
}

var marketOrderFilters = func(req *orderValues) error {
	s := req.symbol
	// PRICE
	if pf := s.PriceFilter(); pf != nil {
		price, err := priceFilter(pf, req.price)
		if err != nil {
			return err
		}

		req.price = price
	}

	// MARKET LOT SIZE
	if lsf := s.MarketLotSizeFilter(); lsf != nil {
		qty, err := marketLotSizeFilter(lsf, req.baseQuantity)
		if err != nil {
			return err
		}

		req.baseQuantity = qty
	}

	// MIN NOTIONAL
	if mnf := s.MinNotionalFilter(); mnf != nil {
		err := minNotionalFilter(mnf, req.price, req.baseQuantity)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyFilters(or *orderValues) error {
	filterFunc, ok := orderTypeFilters[futures.OrderType(or.orderType)]
	if !ok {
		return fmt.Errorf("unsupported order type: %v", or.orderType)
	}

	return filterFunc(or)
}

// priceFilter returns a price adjusted for the tickSize for a given symbol,
// returns an error if the price exceeds min or max value.
func priceFilter(pf *futures.PriceFilter, price float64) (float64, error) {
	tickSize, err := strconv.ParseFloat(pf.TickSize, 64)
	if err != nil {
		return 0, err
	}

	newPrice := price

	if tickSize != 0 {
		// set price to nearest multiple of tickSize
		decimals := stringDecimalPlacesExp(pf.TickSize)
		newPrice = math.Round(price/tickSize) * tickSize
		newPrice = math.Round(newPrice*decimals) / decimals
	}

	minPrice, err := strconv.ParseFloat(pf.MinPrice, 64)
	if err != nil {
		return 0, err
	}

	// reject if price is lower than min price
	if minPrice != 0 && newPrice < minPrice {
		return 0, nil
	}

	// reject is price is higher than max price
	maxPrice, err := strconv.ParseFloat(pf.MaxPrice, 64)
	if err != nil {
		return 0, err
	}

	if maxPrice != 0 && newPrice > maxPrice {
		return 0, nil
	}

	return newPrice, nil
}

func lotSizeFilter(lsf *futures.LotSizeFilter, qty float64) (float64, error) {
	stepSize, err := strconv.ParseFloat(lsf.StepSize, 64)
	if err != nil {
		return 0, err
	}

	decimals := stringDecimalPlacesExp(lsf.StepSize)
	newQty := math.Floor(qty/stepSize) * stepSize
	newQty = math.Round(newQty*decimals) / decimals

	minQty, err := strconv.ParseFloat(lsf.MinQuantity, 64)
	if err != nil {
		return 0, err
	}

	if newQty < minQty {
		return 0, errors.New("quantity too small")
	}

	maxQty, err := strconv.ParseFloat(lsf.MaxQuantity, 64)
	if err != nil {
		return 0, err
	}

	if newQty > maxQty {
		return 0, errors.New("quantity too large")
	}

	return newQty, nil
}

func marketLotSizeFilter(lsf *futures.MarketLotSizeFilter, qty float64) (float64, error) {
	stepSize, err := strconv.ParseFloat(lsf.StepSize, 64)
	if err != nil {
		return 0, err
	}

	decimals := stringDecimalPlacesExp(lsf.StepSize)
	newQty := math.Floor(qty/stepSize) * stepSize
	newQty = math.Round(newQty*decimals) / decimals

	minQty, err := strconv.ParseFloat(lsf.MinQuantity, 64)
	if err != nil {
		return 0, err
	}

	if newQty < minQty {
		return 0, errors.New("quantity too small")
	}

	maxQty, err := strconv.ParseFloat(lsf.MaxQuantity, 64)
	if err != nil {
		return 0, err
	}

	if newQty > maxQty {
		return 0, errors.New("quantity too large")
	}

	return newQty, nil
}

func minNotionalFilter(mnf *futures.MinNotionalFilter, price, qty float64) error {
	if mnf.Notional == "" {
		return nil
	}

	minNotional, err := strconv.ParseFloat(mnf.Notional, 64)
	if err != nil {
		return err
	}

	if price*qty < minNotional {
		return fmt.Errorf("minNotional too small, expected > %f, got %f", minNotional, price*qty)
	}

	return nil
}
