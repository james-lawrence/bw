package iox

import (
	"io"
	"time"
)

// StallWriter closes the underlying writer with the provided error when no write has completed within
// the rolling duration d.
// the timer is paused while a write is in progress, so a slow consumer is not mistaken for a stalled producer.
type StallWriter struct {
	w      io.Writer
	d      time.Duration
	t      *time.Timer
	cancel func(error)
}

// NewStallWriter wraps w, the timer starts immediately. on stall the writer is closed with cause (see CloseWithError).
// cancel (optional, e.g. a context.CancelCauseFunc) is always called when the writer is closed, with the
// error it was closed with. call Close when finished.
func NewStallWriter(w io.Writer, d time.Duration, cause error, cancel func(error)) *StallWriter {
	t := &StallWriter{w: w, d: d, cancel: cancel}
	t.t = time.AfterFunc(d, func() {
		t.CloseWithError(cause)
	})
	return t
}

func (t *StallWriter) Write(p []byte) (n int, err error) {
	t.t.Stop()
	defer t.t.Reset(t.d)
	return t.w.Write(p)
}

// Close stops the stall detection and closes the underlying writer if it is an io.Closer.
func (t *StallWriter) Close() error {
	return t.CloseWithError(nil)
}

// CloseWithError stops the stall detection, calls cancel with the given error and closes the underlying
// writer with it if it supports it (io.PipeWriter), otherwise falls back to io.Closer.
func (t *StallWriter) CloseWithError(err error) error {
	t.t.Stop()

	if t.cancel != nil {
		t.cancel(err)
	}

	switch c := t.w.(type) {
	case interface{ CloseWithError(error) error }:
		return c.CloseWithError(err)
	case io.Closer:
		return c.Close()
	default:
		return nil
	}
}
