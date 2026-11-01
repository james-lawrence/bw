package logx

import (
	"fmt"
	"log"

	"github.com/james-lawrence/bw/internal/errorsx"
	"golang.org/x/time/rate"
)

// DefaultWarnRate is the default rate limit for repeated warnings, one every 10 seconds.
const DefaultWarnRate = rate.Limit(0.1)

// Verbose include stack trace
func Verbose(err error) error {
	if err != nil {
		errorsx.Log(log.Output(2, fmt.Sprintf("%+v\n", err)))
	}

	return err
}

// Limited is a logger that drops messages the limiter denies.
type Limited struct {
	l *rate.Limiter
}

// Limit creates a logger whose output is restricted by the limiter.
func Limit(l *rate.Limiter) Limited {
	return Limited{l: l}
}

// Printf logs the message if the limiter allows it.
func (t Limited) Printf(format string, v ...interface{}) {
	if !t.l.Allow() {
		return
	}

	log.Printf(format, v...)
}
