package timex

import "time"

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
