package iox_test

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/stretchr/testify/require"
)

const errStalled = errorsx.String("stalled")

func TestStallWriter(t *testing.T) {
	t.Run("fires when writes stop", func(t *testing.T) {
		stalled := make(chan struct{}, 2)
		w := iox.NewStallWriter(io.Discard, 20*time.Millisecond, errStalled, func(err error) {
			if errors.Is(err, errStalled) {
				stalled <- struct{}{}
			}
		})
		defer w.Close()

		_, err := w.Write([]byte("a"))
		require.NoError(t, err)

		select {
		case <-stalled:
		case <-time.After(time.Second):
			t.Fatal("expected stall")
		}
	})

	t.Run("does not fire while writes progress", func(t *testing.T) {
		w := iox.NewStallWriter(io.Discard, 50*time.Millisecond, errStalled, func(err error) {
			if errors.Is(err, errStalled) {
				t.Error("unexpected stall")
			}
		})
		defer w.Close()

		for range 5 {
			time.Sleep(20 * time.Millisecond)
			_, err := w.Write([]byte("a"))
			require.NoError(t, err)
		}
	})

	t.Run("closes the pipe with the error", func(t *testing.T) {
		pr, pw := io.Pipe()
		w := iox.NewStallWriter(pw, 20*time.Millisecond, errStalled, nil)
		defer w.Close()

		_, err := pr.Read(make([]byte, 1))
		require.ErrorIs(t, err, errStalled)
	})

	t.Run("cancel is called on close", func(t *testing.T) {
		var got error
		w := iox.NewStallWriter(io.Discard, time.Minute, errStalled, func(err error) { got = err })
		require.NoError(t, w.CloseWithError(io.ErrClosedPipe))
		require.ErrorIs(t, got, io.ErrClosedPipe)
	})
}
