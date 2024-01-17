package geometry_test

import (
	"testing"
	"time"

	"github.com/H3Cki/Plotrader/core/domain/geometry"
	"github.com/stretchr/testify/assert"
)

func TestSchedule_At(t *testing.T) {
	type fields struct {
		Since time.Time
		Until time.Time
		Plot  geometry.Plot
	}

	tests := []struct {
		name        string
		fields      fields
		t           time.Time
		want        float64
		expectedErr error
	}{
		{
			name:   "always valid - zero time",
			fields: fields{Plot: &alwaysValid{5}},
			t:      time.Time{},
			want:   5,
		},
		{
			name:   "always valid - max time",
			fields: fields{Plot: &alwaysValid{5}},
			t:      time.Unix(1<<63-1, 0),
			want:   5,
		},
		{
			name:   "since - exact",
			fields: fields{Since: time.Unix(10, 0), Plot: &alwaysValid{5}},
			t:      time.Unix(10, 0),
			want:   5,
		},
		{
			name:   "since - after",
			fields: fields{Since: time.Unix(10, 0), Plot: &alwaysValid{5}},
			t:      time.Unix(11, 0),
			want:   5,
		},
		{
			name:        "since - before",
			fields:      fields{Since: time.Unix(10, 0), Plot: &alwaysValid{5}},
			t:           time.Unix(9, 0),
			want:        0,
			expectedErr: geometry.ErrPlotOutOfRange,
		},
		{
			name:        "until - exact",
			fields:      fields{Until: time.Unix(10, 0), Plot: &alwaysValid{5}},
			t:           time.Unix(10, 0),
			want:        0,
			expectedErr: geometry.ErrPlotOutOfRange,
		},
		{
			name:        "until - after",
			fields:      fields{Until: time.Unix(10, 0), Plot: &alwaysValid{}},
			t:           time.Unix(11, 0),
			want:        0,
			expectedErr: geometry.ErrPlotOutOfRange,
		},
		{
			name:   "until - before",
			fields: fields{Until: time.Unix(10, 0), Plot: &alwaysValid{5}},
			t:      time.Unix(9, 0),
			want:   5,
		},
		{
			name:   "since, until - exact since",
			fields: fields{Since: time.Unix(10, 0), Until: time.Unix(20, 0), Plot: &alwaysValid{5}},
			t:      time.Unix(10, 0),
			want:   5,
		},
		{
			name:        "since, until - exact until",
			fields:      fields{Since: time.Unix(10, 0), Until: time.Unix(20, 0), Plot: &alwaysValid{5}},
			t:           time.Unix(20, 0),
			want:        0,
			expectedErr: geometry.ErrPlotOutOfRange,
		},
		{
			name:   "since, until - in between",
			fields: fields{Since: time.Unix(10, 0), Until: time.Unix(20, 0), Plot: &alwaysValid{5}},
			t:      time.Unix(15, 0),
			want:   5,
		},
		{
			name:        "valid since, until - before",
			fields:      fields{Since: time.Unix(10, 0), Until: time.Unix(20, 0), Plot: &alwaysValid{5}},
			t:           time.Unix(9, 0),
			want:        0,
			expectedErr: geometry.ErrPlotOutOfRange,
		},
		{
			name:        "valid since, until - after",
			fields:      fields{Since: time.Unix(10, 0), Until: time.Unix(20, 0), Plot: &alwaysValid{5}},
			t:           time.Unix(9, 0),
			want:        0,
			expectedErr: geometry.ErrPlotOutOfRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &geometry.Limit{
				From: tt.fields.Since,
				To:   tt.fields.Until,
				Plot: tt.fields.Plot,
			}
			got, err := v.At(tt.t)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

type alwaysValid struct {
	returns float64
}

func (v *alwaysValid) At(time.Time) (float64, error) {
	return v.returns, nil
}
