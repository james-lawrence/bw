package iox_test

import (
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/james-lawrence/bw/internal/iox"
)

func TestCountingReader(t *testing.T) {
	var n atomic.Uint64
	if _, err := io.Copy(io.Discard, iox.ReaderCounting(strings.NewReader("hello world"), &n)); err != nil {
		t.Fatal(err)
	}
	if got := n.Load(); got != 11 {
		t.Fatalf("expected 11 bytes got %d", got)
	}
}
