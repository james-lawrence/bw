package timex

import "time"

// Every executes the provided function every duration.
func Every(d time.Duration, do func()) {
	for _ = range time.Tick(d) {
		do()
	}
}
