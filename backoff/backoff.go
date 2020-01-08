package backoff

import (
	"math"
	"math/rand"
	"time"
)

// Strategy strategy to compute how long to wait before retrying message.
type Strategy interface {
	Backoff(attempt int) time.Duration
}

// Maximum sets an upper bound for the strategy.
func Maximum(d time.Duration, s Strategy) Strategy {
	return StrategyFunc(func(attempt int) time.Duration {
		if x := s.Backoff(attempt); x < d {
			return x
		}

		return d
	})
}

// Jitter set a jitter frame for the strategy.
// rounds to 250 milliseconds
func Jitter(maximum time.Duration, s Strategy) Strategy {
	return StrategyFunc(func(attempt int) time.Duration {
		x := s.Backoff(attempt)
		return x + time.Duration(rand.Intn(int(maximum)))
	})
}

// StrategyFunc convience helper to convert a pure function into a backoff strategy.
type StrategyFunc func(attempt int) time.Duration

// Backoff implements Strategy
func (t StrategyFunc) Backoff(attempt int) time.Duration {
	return t(attempt)
}

// Constant always returns the provided duration regardless of the attempt.
func Constant(d time.Duration) Strategy {
	return StrategyFunc(func(attempt int) time.Duration {
		return d
	})
}

type exponential struct {
	scale time.Duration
}

func (t *exponential) Backoff(attempt int) time.Duration {
	// if the exponential wraps around fall back to using maximum.
	if exp := time.Duration(1 << uint(attempt)); exp > 0 {
		return exp * t.scale
	}

	return time.Duration(math.MaxInt64)
}

// Exponential implements expoential backoff.
func Exponential(scale time.Duration) Strategy {
	if scale == 0 {
		panic("exponential backoff can't be scaled by 0")
	}
	return &exponential{
		scale: scale,
	}
}

// Explicit an explicit set of delays to use. if the attempt is larger than
// the number of values it restarts at the first delay.
func Explicit(delays ...time.Duration) Strategy {
	return explicit{delays: delays}
}

type explicit struct {
	delays []time.Duration
}

func (t explicit) Backoff(attempt int) time.Duration {
	return t.delays[attempt%len(t.delays)]
}
