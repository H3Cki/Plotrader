package geometry_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
	"github.com/stretchr/testify/assert"
)

func TestNewMin(t *testing.T) {
	tests := []struct {
		plots     []geometry.Plot
		expectErr bool
	}{
		{nil, true},
		{[]geometry.Plot{}, true},
		{[]geometry.Plot{&geometry.Line{A: 0, B: 1}}, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(len(tt.plots)), func(t *testing.T) {
			m, err := geometry.NewMin(tt.plots)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, m)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, m)
			}
		})
	}
}

func TestNewMax(t *testing.T) {
	tests := []struct {
		plots     []geometry.Plot
		expectErr bool
	}{
		{nil, true},
		{[]geometry.Plot{}, true},
		{[]geometry.Plot{&geometry.Line{A: 0, B: 1}}, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(len(tt.plots)), func(t *testing.T) {
			m, err := geometry.NewMax(tt.plots)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, m)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, m)
			}
		})
	}
}

func TestMin_At(t *testing.T) {
	tests := []struct {
		name        string
		plots       []geometry.Plot
		t           time.Time
		want        float64
		inRange     bool
		expectedErr error
	}{
		{
			name: "invalid, 2 valid, invalid",
			plots: []geometry.Plot{
				&neverValid{5},
				&geometry.Line{A: 0, B: 10},
				&geometry.Line{A: 0, B: 15},
				&neverValid{15},
			},
			t:       time.Time{},
			want:    10,
			inRange: true,
		},
		{
			name: "1 until, 1.5 after - until",
			plots: []geometry.Plot{
				&geometry.Limit{To: time.Unix(10, 0), Plot: &geometry.Line{B: 1}},
				&geometry.Line{B: 1.5},
			},
			t:       time.Unix(9, 0),
			want:    1,
			inRange: true,
		},
		{
			name: "1 until, 1.5 after - after",
			plots: []geometry.Plot{
				&geometry.Limit{To: time.Unix(10, 0), Plot: &geometry.Line{B: 1}},
				&geometry.Line{B: 1.5},
			},
			t:       time.Unix(10, 0),
			want:    1.5,
			inRange: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &geometry.Min{
				Plots: tt.plots,
			}

			got, err := m.At(tt.t)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMax_At(t *testing.T) {
	tests := []struct {
		name        string
		plots       []geometry.Plot
		t           time.Time
		want        float64
		inRange     bool
		expectedErr error
	}{
		{
			name: "invalid, 2 valid, invalid",
			plots: []geometry.Plot{
				&neverValid{5},
				&geometry.Line{A: 0, B: 10},
				&geometry.Line{A: 0, B: 15},
				&neverValid{15},
			},
			t:       time.Time{},
			want:    15,
			inRange: true,
		},
		{
			name: "2 until, 1 after - until",
			plots: []geometry.Plot{
				&geometry.Limit{To: time.Unix(10, 0), Plot: &geometry.Line{B: 2}},
				&geometry.Line{B: 1},
			},
			t:       time.Unix(9, 0),
			want:    2,
			inRange: true,
		},
		{
			name: "2 until, 1 after - after",
			plots: []geometry.Plot{
				&geometry.Limit{To: time.Unix(10, 0), Plot: &geometry.Line{B: 2}},
				&geometry.Line{B: 1},
			},
			t:       time.Unix(10, 0),
			want:    1,
			inRange: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &geometry.Max{
				Plots: tt.plots,
			}

			got, err := m.At(tt.t)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

type neverValid struct {
	returns float64
}

func (v *neverValid) At(time.Time) (float64, error) {
	return v.returns, geometry.ErrPlotOutOfRange
}
