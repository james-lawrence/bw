package iox

import (
	"io"
	"sync/atomic"
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

// StalledReader closes the underlying reader with the provided error when a read has not completed within d.
// the timer only runs while a read is in progress, so a slow consumer is not mistaken for a stalled producer.
type StalledReader struct {
	r       io.Reader
	d       time.Duration
	cause   error
	cancel  func(error)
	stalled atomic.Bool
}

// NewStalledReader wraps r. on stall r is closed with cause (see CloseWithError) and the read returns cause.
// cancel (optional) is always called when the reader is closed, with the error it was closed with.
// call Close when finished.
func NewStalledReader(r io.Reader, d time.Duration, cause error, cancel func(error)) *StalledReader {
	return &StalledReader{r: r, d: d, cause: cause, cancel: cancel}
}

func (t *StalledReader) Read(p []byte) (n int, err error) {
	timer := time.AfterFunc(t.d, func() {
		t.stalled.Store(true)
		t.CloseWithError(t.cause)
	})
	n, err = t.r.Read(p)
	timer.Stop()

	if n == 0 && t.stalled.Load() {
		return 0, t.cause
	}

	return n, err
}

// Close closes the underlying reader if it is an io.Closer.
func (t *StalledReader) Close() error {
	return t.CloseWithError(nil)
}

// CloseWithError calls cancel with the given error and closes the underlying reader with it
// if it supports it, otherwise falls back to io.Closer.
func (t *StalledReader) CloseWithError(err error) error {
	if t.cancel != nil {
		t.cancel(err)
	}

	switch c := t.r.(type) {
	case interface{ CloseWithError(error) error }:
		return c.CloseWithError(err)
	case io.Closer:
		return c.Close()
	default:
		return nil
	}
}
