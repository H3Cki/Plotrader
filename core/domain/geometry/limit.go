package geometry

import (
	"time"
)

// Limit is a Plot wrapper that allows it to be valid only in certain periods of time.
// The range is [From, To), time constraints are considered active only when they are non-zero,
// meaning that if Since is zero and Until is not, only Until constraint will be applied.
type Limit struct {
	Plot     Plot
	From, To time.Time
}

// NewLimit is a constructor for Schedule, sets since and until times to UTC timezone
func NewLimit(plot Plot, from, to time.Time) *Limit {
	return &Limit{Plot: plot, From: from, To: to}
}

func (v *Limit) At(t time.Time) (float64, error) {
	if !v.inRange(t) {
		return 0, ErrPlotOutOfRange
	}
	return v.Plot.At(t)
}

func (v *Limit) inRange(t time.Time) bool {
	return (!t.Before(v.From) || v.From.IsZero()) && (t.Before(v.To) || v.To.IsZero())
}
