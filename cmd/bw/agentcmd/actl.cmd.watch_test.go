package agentcmd_test

import (
	"testing"
	"time"

	"github.com/james-lawrence/bw/agenttestutil"
)

// watch streams events until its context ends, it must connect and keep running.
func TestWatchStaysConnected(t *testing.T) {
	boot(t)

	h := actl(t)
	result := make(chan error, 1)
	go func() { result <- h.Run("watch", agenttestutil.Environment, "--insecure") }()

	select {
	case err := <-result:
		t.Fatalf("watch returned early: %v", err)
	case <-time.After(2 * time.Second):
	}
}
