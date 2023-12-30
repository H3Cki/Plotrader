package geometry_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
	"github.com/stretchr/testify/assert"
)

func TestAbsoluteOffset_Offset(t *testing.T) {
	tests := []struct {
		pad, v, result float64
	}{
		{10, 5, 15},
		{-10, 5, -5},
		{0, 5, 5},
	}

	for _, test := range tests {
		offsetter := geometry.AbsoluteOffset{Value: test.pad}

		assert.Equal(t, test.result, offsetter.Offset(test.v))
	}
}

func TestPercentageOffset_Offset(t *testing.T) {
	tests := []struct {
		pad, v, result float64
	}{
		{-0.5, 5, 2.5},
		{0.5, 5, 7.5},
		{0, 5, 5},
	}

	for _, test := range tests {
		offsetter := geometry.PercentageOffset{Percentage: test.pad}

		assert.Equal(t, test.result, offsetter.Offset(test.v))
	}
}

func TestNewPercentageOffset(t *testing.T) {
	e, _ := time.ParseDuration("4h")

	now := time.Now()

	dur := e.Seconds()
	div := int64(float64(now.Unix()) / dur)

	previousStart := div * int64(dur)
	nextStart := previousStart + int64(dur)

	fmt.Printf("Previous: 	%s\nNow: 		%s\nNext:	 	%s\n", time.Unix(previousStart, 0).In(time.UTC).String(), now.String(), time.Unix(nextStart, 0).In(time.UTC).String())
}
