package timex_test

import (
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/timex"
)

func TestFirstNonZero(t *testing.T) {
	a := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	b := a.Add(time.Hour)

	for name, tc := range map[string]struct {
		in       []time.Time
		expected time.Time
	}{
		"empty":          {},
		"all zero":       {in: []time.Time{{}, {}}},
		"first":          {in: []time.Time{a, b}, expected: a},
		"skips zero":     {in: []time.Time{{}, b, a}, expected: b},
		"trailing value": {in: []time.Time{{}, {}, a}, expected: a},
	} {
		t.Run(name, func(t *testing.T) {
			if got := timex.FirstNonZero(tc.in...); !got.Equal(tc.expected) {
				t.Fatalf("expected %v got %v", tc.expected, got)
			}
		})
	}
}

func TestUnixOrZero(t *testing.T) {
	if got := timex.UnixOrZero(0); !got.IsZero() {
		t.Fatalf("expected the zero time for 0 got %v", got)
	}

	if got := timex.UnixOrZero(-5); !got.IsZero() {
		t.Fatalf("expected the zero time for a negative value got %v", got)
	}

	if got := timex.UnixOrZero(1793512800); !got.Equal(time.Unix(1793512800, 0)) {
		t.Fatalf("unexpected time %v", got)
	}
}
