package backoff

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testBackoff(t *testing.T, attempts int, s Strategy, expected ...time.Duration) {
	for i := 0; i < attempts; i++ {
		require.Equal(t, expected[i], s.Backoff(i), fmt.Sprintf("attempt %d", i))
	}
}

func expectedDurationTest(t *testing.T, attempt int, s Strategy, expected time.Duration) {
	require.Equal(t, expected, s.Backoff(attempt))
}

func TestExplicit(t *testing.T) {
	t.Run("more attempts than delays", func(t *testing.T) {
		testBackoff(t, 5, Explicit(1*time.Second, 2*time.Second, 3*time.Second), 1*time.Second, 2*time.Second, 3*time.Second, 1*time.Second, 2*time.Second)
	})
}

func TestExponential(t *testing.T) {
	t.Run("should double each time", func(t *testing.T) {
		testBackoff(t, 5, Exponential(1*time.Second), 1*time.Second, 2*time.Second, 4*time.Second, 8*time.Second, 16*time.Second)
	})

	t.Run("should gracefully handle overflows", func(t *testing.T) {
		// 101 attempts: doubling for the first 34, then saturated at MaxInt64.
		expected := make([]time.Duration, 0, 101)
		for i := 0; i < 34; i++ {
			expected = append(expected, time.Second<<uint(i))
		}
		for i := 34; i < 101; i++ {
			expected = append(expected, time.Duration(math.MaxInt64))
		}
		testBackoff(t, 101, Exponential(1*time.Second), expected...)
	})
}

func TestConstant(t *testing.T) {
	t.Run("should remain constant", func(t *testing.T) {
		testBackoff(t, 5, Constant(1*time.Second), 1*time.Second, 1*time.Second, 1*time.Second, 1*time.Second, 1*time.Second)
	})
}

func TestExponentialBackoff(t *testing.T) {
	for _, c := range []struct {
		name     string
		attempt  int
		s        Strategy
		expected time.Duration
	}{
		{"attempt 0", 0, Exponential(1 * time.Second), 1 * time.Second},
		{"attempt 1", 1, Exponential(1 * time.Second), 2 * time.Second},
		{"attempt 2", 2, Exponential(1 * time.Second), 4 * time.Second},
		{"attempt 3", 3, Exponential(1 * time.Second), 8 * time.Second},
		{"attempt 36", 36, Exponential(1 * time.Second), time.Duration(math.MaxInt64)},
		{"attempt 37", 37, Exponential(1 * time.Second), time.Duration(math.MaxInt64)},
		{"attempt 54 - overflow", 54, Exponential(1 * time.Second), time.Duration(math.MaxInt64)},
		{"with scaling - attempt 0", 0, Exponential(500 * time.Millisecond), 500 * time.Millisecond},
		{"with scaling - attempt 1", 1, Exponential(500 * time.Millisecond), 1 * time.Second},
		{"with scaling - attempt 2", 2, Exponential(500 * time.Millisecond), 2 * time.Second},
		{"with scaling - attempt 3", 3, Exponential(500 * time.Millisecond), 4 * time.Second},
		{"max attempt value", math.MaxInt64, Exponential(1 * time.Second), time.Duration(math.MaxInt64)},
	} {
		t.Run(c.name, func(t *testing.T) {
			expectedDurationTest(t, c.attempt, c.s, c.expected)
		})
	}
}

func TestJitter(t *testing.T) {
	t.Run("example 1 - with jitter", func(t *testing.T) {
		expectedDurationTest(t, 57, New(Exponential(1*time.Second), Jitter(0.25)), time.Duration(math.MaxInt64))
	})
}
