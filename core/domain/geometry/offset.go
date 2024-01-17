package geometry

import "time"

type Offsetter interface {
	Offset(v float64) float64
}

// AbsoluteOffset offsets the value by another value
type AbsoluteOffset struct {
	Value float64
}

func NewAbsoluteOffset(value float64) *AbsoluteOffset {
	return &AbsoluteOffset{value}
}

func (p *AbsoluteOffset) Offset(v float64) float64 {
	return v + p.Value
}

// PercentageOffset offsets the value by a percentage of it
type PercentageOffset struct {
	Percentage float64
}

func NewPercentageOffset(value float64) *PercentageOffset {
	return &PercentageOffset{value}
}

func (p *PercentageOffset) Offset(v float64) float64 {
	return v + (v * p.Percentage)
}

// OffsetPlot is a plot wrapper that allows it to be offset seamlesly
type OffsetPlot struct {
	Offsetter Offsetter
	Plot      Plot
}

func NewOffsetPlot(plot Plot, offset Offsetter) *OffsetPlot {
	return &OffsetPlot{
		Offsetter: offset,
		Plot:      plot,
	}
}

func (o *OffsetPlot) At(t time.Time) (float64, error) {
	v, err := o.Plot.At(t)
	if err != nil {
		return 0, err
	}

	return o.Offsetter.Offset(v), nil
}
