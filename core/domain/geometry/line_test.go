package geometry_test

import (
	"testing"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
	"github.com/stretchr/testify/assert"
)

func TestLine_At(t *testing.T) {
	tests := []struct {
		name        string
		p1, p2      geometry.Point
		x           time.Time
		y           float64
		newErr      error
		expectedErr error
	}{
		{
			p1: geometry.Point{time.Unix(0, 0), 0},
			p2: geometry.Point{time.Unix(1, 0), 1},
			x:  time.Unix(2, 0),
			y:  2,
		},
		{
			p1: geometry.Point{time.Unix(0, 0), 0},
			p2: geometry.Point{time.Unix(1, 0), 1},
			x:  time.Unix(-500, 0),
			y:  -500,
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			line, err := geometry.NewLine(tt.p1, tt.p2)
			assert.ErrorIs(t, err, tt.newErr)

			y, err := line.At(tt.x)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.y, y)
		})
	}
}

func TestLogLine_At(t *testing.T) {
	tests := []struct {
		p1, p2       geometry.Point
		x            time.Time
		y            float64
		expectNewErr bool
		expectedErr  error
	}{
		{
			p1: geometry.Point{time.Unix(0, 0), 1},
			p2: geometry.Point{time.Unix(2, 0), 100},
			x:  time.Unix(0, 0),
			y:  1,
		},
		{
			p1: geometry.Point{time.Unix(0, 0), 1},
			p2: geometry.Point{time.Unix(2, 0), 100},
			x:  time.Unix(3, 0),
			y:  1000,
		},
		{
			p1: geometry.Point{time.Unix(0, 0), 1},
			p2: geometry.Point{time.Unix(2, 0), 100},
			x:  time.Unix(2, 0),
			y:  100,
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			line, err := geometry.NewLogLine(tt.p1, tt.p2)
			assert.True(t, (err != nil) == tt.expectNewErr)

			y, err := line.At(tt.x)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.y, y)
		})
	}
}
