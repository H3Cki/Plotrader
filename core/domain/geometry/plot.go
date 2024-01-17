package geometry

import (
	"sort"
	"time"

	"github.com/pkg/errors"
)

var (
	ErrPlotOutOfRange = errors.New("plot out of range")
)

type PlotSpec map[string]any

func (p PlotSpec) Parse() (Plot, error) {
	return parsePlotMap(p)
}

type Plot interface {
	At(time.Time) (float64, error)
}

type Point struct {
	Date  time.Time
	Price float64
}

func timeToFloat64(t time.Time) float64 {
	return float64(t.Unix())
}

func sortPoints(points ...Point) []Point {
	sort.Slice(points, func(i, j int) bool { return points[i].Date.Before(points[j].Date) })
	return points
}
