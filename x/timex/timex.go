package timex

import (
	"math"
	"time"
)

// Every executes the provided function every duration.
func Every(d time.Duration, do func()) {
	for _ = range time.Tick(d) {
		do()
	}
}

// DurationOrDefault ...
func DurationOrDefault(a, b time.Duration) time.Duration {
	if a == 0 {
		return b
	}
	return a
}

// DurationMax select the maximum duration from the set.
func DurationMax(ds ...time.Duration) (d time.Duration) {
	for _, c := range ds {
		if c > d {
			d = c
		}
	}

	return d
}

// DurationMin select the minimum duration from the set.
func DurationMin(ds ...time.Duration) (d time.Duration) {
	d = math.MaxInt64

	for _, c := range ds {
		if c < d {
			d = c
		}
	}

	return d
}
