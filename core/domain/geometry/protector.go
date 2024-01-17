package geometry

import (
	"errors"
	"fmt"
	"math"
	"time"
)

var ErrPriceProtection = errors.New("price protection")

// Protector checks for most obvious wrong price values like:
// -Inf, 0, +Inf and returns an error if such price is calculated
type Protector struct {
	of Plot
}

func NewProtector(of Plot) Plot {
	return &Protector{of: of}
}

func (p *Protector) At(t time.Time) (float64, error) {
	price, err := p.of.At(t)
	if err != nil {
		return 0, err
	}
	if price == 0 || math.IsInf(price, 0) {
		return 0, fmt.Errorf("%w: %f", ErrPriceProtection, price)
	}
	return price, err
}
