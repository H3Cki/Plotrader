package geometry

import (
	"errors"
	"time"
)

// Min is a plot aggregator which returns the minimum value returned from all the plots.
// Min is valid when at least one plot is valid at a given time.
type Min struct {
	Plots []Plot
}

// NewMin is a constructor for Min aggregator, returns error if provided plot list is empty
func NewMin(plots []Plot) (*Min, error) {
	if len(plots) == 0 {
		return nil, errors.New("error creating min aggregator: empty plot list")
	}

	return &Min{Plots: plots}, nil
}

func (m *Min) At(t time.Time) (float64, error) {
	var max *float64

	for _, p := range m.Plots {
		plotAt, err := p.At(t)
		if errors.Is(err, ErrPlotOutOfRange) {
			continue
		}
		if err != nil {
			return 0, err
		}

		if max == nil || plotAt < *max {
			max = &plotAt
		}
	}

	if max == nil {
		return 0, ErrPlotOutOfRange
	}

	return *max, nil
}

// Max is a plot aggregator which returns the maximum value returned from all the plots.
// Max is valid when at least one plot is valid at a given time.
type Max struct {
	Plots []Plot
}

// NewMax is a constructor for Min aggregator, returns error if provided plot list is empty
func NewMax(plots []Plot) (*Max, error) {
	if len(plots) == 0 {
		return nil, errors.New("error creating max aggregator: empty plot list")
	}

	return &Max{Plots: plots}, nil
}

func (m *Max) At(t time.Time) (float64, error) {
	var max *float64

	for _, p := range m.Plots {
		plotAt, err := p.At(t)
		if errors.Is(err, ErrPlotOutOfRange) {
			continue
		}
		if err != nil {
			return 0, err
		}

		if max == nil || plotAt > *max {
			max = &plotAt
		}
	}

	if max == nil {
		return 0, ErrPlotOutOfRange
	}

	return *max, nil
}
