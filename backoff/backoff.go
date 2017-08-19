package backoff

import "time"

// Strategy strategy to compute how long to wait before retrying message.
type Strategy interface {
	Backoff(attempt int) time.Duration
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
	current time.Duration
}

func (t *exponential) Backoff(attempt int) time.Duration {
	d := t.current
	t.current = t.current * 2
	return d
}

// Exponential implements expoential backoff.
func Exponential(initial time.Duration) Strategy {
	if initial == 0 {
		panic("exponential backoff can't start at 0")
	}
	return &exponential{
		current: initial,
	}
}

// Maximum - applies a maximum backoff to a strategy.
func Maximum(max time.Duration, s Strategy) Strategy {
	return StrategyFunc(func(attempt int) time.Duration {
		computed := s.Backoff(attempt)
		if max < computed {
			return max
		}
		return computed
	})
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
