package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_computeTimeOnlyDelta(t *testing.T) {
	tests := []struct {
		name     string
		previous time.Duration
		current  time.Duration
		want     time.Duration
	}{
		{
			name:     "simple forward delta",
			previous: 10 * time.Hour,
			current:  11 * time.Hour,
			want:     1 * time.Hour,
		},
		{
			name:     "forward delta with minutes and seconds",
			previous: 19*time.Hour + 25*time.Minute,
			current:  19*time.Hour + 30*time.Minute + 15*time.Second,
			want:     5*time.Minute + 15*time.Second,
		},
		{
			name:     "forward delta with milliseconds",
			previous: 19*time.Hour + 25*time.Minute + 0*time.Second + 949*time.Millisecond,
			current:  19*time.Hour + 25*time.Minute + 1*time.Second + 100*time.Millisecond,
			want:     151 * time.Millisecond,
		},
		{
			name:     "rollover at midnight - simple case",
			previous: 23 * time.Hour,
			current:  0 * time.Hour,
			want:     1 * time.Hour,
		},
		{
			name:     "rollover at midnight - with minutes",
			previous: 23*time.Hour + 59*time.Minute,
			current:  0*time.Hour + 1*time.Minute,
			want:     2 * time.Minute,
		},
		{
			name:     "rollover at midnight - realistic example",
			previous: 23*time.Hour + 59*time.Minute + 59*time.Second,
			current:  0*time.Hour + 0*time.Minute + 1*time.Second,
			want:     2 * time.Second,
		},
		{
			name:     "rollover with large gap",
			previous: 23 * time.Hour,
			current:  6 * time.Hour,
			want:     7 * time.Hour,
		},
		{
			name:     "same time - zero delta",
			previous: 15 * time.Hour,
			current:  15 * time.Hour,
			want:     0,
		},
		{
			name:     "midnight to midnight",
			previous: 0,
			current:  0,
			want:     0,
		},
		{
			name:     "near end of day to near start",
			previous: 23*time.Hour + 55*time.Minute + 30*time.Second + 500*time.Millisecond,
			current:  0*time.Hour + 5*time.Minute + 30*time.Second + 600*time.Millisecond,
			want:     10*time.Minute + 100*time.Millisecond,
		},
		{
			name:     "multiple hours forward within same day",
			previous: 8 * time.Hour,
			current:  17 * time.Hour,
			want:     9 * time.Hour,
		},
		{
			name:     "nanosecond precision",
			previous: 10*time.Hour + 123*time.Nanosecond,
			current:  10*time.Hour + 456*time.Nanosecond,
			want:     333 * time.Nanosecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeTimeOnlyDelta(tt.previous, tt.current)
			assert.Equal(t, tt.want, got, "computeTimeOnlyDelta(%v, %v) = %v, want %v", tt.previous, tt.current, got, tt.want)
		})
	}
}
