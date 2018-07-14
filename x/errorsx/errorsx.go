package errorsx

import "time"

// Compact returns the first error in the set, if any.
func Compact(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

// String useful wrapper for string constants as errors.
type String string

func (t String) Error() string {
	return string(t)
}

// Timeout error.
type Timeout interface {
	Timedout() time.Duration
}

// Timedout represents a timeout.
func Timedout(cause error, d time.Duration) error {
	return timeout{
		error: cause,
		d:     d,
	}
}

type timeout struct {
	error
	d time.Duration
}

func (t timeout) Timedout() time.Duration {
	return t.d
}
