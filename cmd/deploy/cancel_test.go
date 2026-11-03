package deploy_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/stretchr/testify/require"
)

// cancelling a deploy must return as soon as the agent acknowledges the cancel,
// previously it blocked until the process was killed.
func TestCancelReturnsOnceCancelled(t *testing.T) {
	if testing.Short() {
		t.Skip("boots an agent, skipping in short mode")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().(*net.TCPAddr)
	require.NoError(t, l.Close())

	require.NoError(t, agenttestutil.Agent(t, addr))
	require.NoError(t, agenttestutil.Client(t, addr))
	require.NoError(t, agenttestutil.WaitQuorum(context.Background()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- deploy.Cancel(&deploy.Context{
			Environment: agenttestutil.Environment,
			Insecure:    true,
			Context:     ctx,
			CancelFunc:  cancel,
			WaitGroup:   &sync.WaitGroup{},
		})
	}()

	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(20 * time.Second):
		t.Fatal("deploy cancel did not return after the cancel was acknowledged")
	}
}
