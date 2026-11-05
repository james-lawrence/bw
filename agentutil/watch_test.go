package agentutil_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agentutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type failingDialer struct{}

func (failingDialer) DialContext(ctx context.Context, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	return nil, errors.New("unable to dial")
}

func TestWatchEventsShouldReturnOnceTheContextIsCancelledEvenIfEventsAreNoLongerConsumed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// nothing reads from events, e.g. the ux has already exited.
	events := make(chan *agent.Message)
	returned := make(chan struct{})

	go func() {
		defer close(returned)
		WatchEvents(ctx, agent.NewPeer("local"), failingDialer{}, events)
	}()

	// give the watcher time to block delivering the dial failure.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-returned:
	case <-time.After(time.Second):
		require.FailNow(t, "WatchEvents did not return after the context was cancelled")
	}
}
