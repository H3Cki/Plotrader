package geometry

import (
	"errors"
	"math"
	"time"
)

type Line struct {
	A, B float64
}

func NewLine(p0, p1 Point) (*Line, error) {
	if p0.Date.Equal(p1.Date) {
		return nil, errors.New("error creating new line: both points have the same date")
	}

	sorted := sortPoints(p0, p1)
	p0 = sorted[0]
	p1 = sorted[1]

	p0DateFloat := timeToFloat64(p0.Date)
	p1DateFloat := timeToFloat64(p1.Date)

	a := (p1.Price - p0.Price) / (p1DateFloat - p0DateFloat)
	b := p0.Price - (a * p0DateFloat)

	l := &Line{
		A: a,
		B: b,
	}

	return l, nil
}

func (l *Line) At(date time.Time) (float64, error) {
	return l.A*timeToFloat64(date) + l.B, nil
}

// Straight line on semi-logarighmic (x, log10) graph
type LogLine struct {
	M, K, Xoffset float64
}

func NewLogLine(p0, p1 Point) (*LogLine, error) {
	if p0.Date.Equal(p1.Date) {
		return nil, errors.New("error creating new line: both points have the same date")
	}

	sorted := sortPoints(p0, p1)
	p0 = sorted[0]
	p1 = sorted[1]

	xOffset := timeToFloat64(p0.Date)

	x0 := 0.0
	x1 := timeToFloat64(p1.Date) - xOffset

	y0 := p0.Price
	y1 := p1.Price
	m := (math.Log10(y1) - math.Log10(y0)) / (x1 - x0)

	l := &LogLine{
		M:       m,
		K:       y0,
		Xoffset: xOffset,
	}

	return l, nil
}

func (l *LogLine) At(date time.Time) (float64, error) {
	x := timeToFloat64(date) - l.Xoffset
	return l.K * math.Pow(10, l.M*x), nil
}
