package geometry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
	"github.com/stretchr/testify/assert"
)

var testErr = errors.New("test error")

func TestProtector_At(t *testing.T) {
	zero := 0.0
	tests := []struct {
		name        string
		of          geometry.Plot
		want        float64
		expectedErr error
	}{
		{
			name:        "zero protection",
			of:          &valuePlot{0},
			want:        0,
			expectedErr: geometry.ErrPriceProtection,
		},
		{
			name:        "-inf protection",
			of:          &valuePlot{-1 / zero},
			want:        0,
			expectedErr: geometry.ErrPriceProtection,
		},
		{
			name:        "+inf protection",
			of:          &valuePlot{1 / zero},
			want:        0,
			expectedErr: geometry.ErrPriceProtection,
		},
		{
			name:        "no protection",
			of:          &valuePlot{1},
			want:        1,
			expectedErr: nil,
		},
		{
			name:        "propagate error",
			of:          &errPlot{},
			want:        0,
			expectedErr: testErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := geometry.NewProtector(tt.of)
			got, err := p.At(time.Time{})
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

type valuePlot struct {
	ret float64
}

func (t *valuePlot) At(time.Time) (float64, error) {
	return t.ret, nil
}

type errPlot struct{}

func (t *errPlot) At(time.Time) (float64, error) {
	return 0, testErr
}
