package quorum

import (
	"context"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// dispatcher that fails the first n dispatches.
type flakyDispatcher struct {
	m         sync.Mutex
	failures  int
	attempts  int
	delivered []*agent.Message
}

func (t *flakyDispatcher) Dispatch(_ context.Context, ms ...*agent.Message) error {
	t.m.Lock()
	defer t.m.Unlock()

	t.attempts++
	if t.attempts <= t.failures {
		return status.Error(codes.Unavailable, "connection is unavailable")
	}

	t.delivered = append(t.delivered, ms...)
	return nil
}

func TestDispatchDeployResult(t *testing.T) {
	t.Run("should retry until the deploy result is delivered", func(t *testing.T) {
		local := agent.NewPeer("local")
		d := &flakyDispatcher{failures: 1}
		dcmd := agent.DeployCommandDone("tester")

		require.NoError(t, dispatchDeployResult(context.Background(), d, local, dcmd))
		require.Equal(t, 2, d.attempts)
		require.Len(t, d.delivered, 1)
		require.Equal(t, agent.DeployCommand_Done, d.delivered[0].GetDeployCommand().Command)
	})
}

func TestObserver(t *testing.T) {
	t.Run("should journal messages", func(t *testing.T) {
		d := &flakyDispatcher{}
		m := agent.LogInfo(agent.NewPeer("local"), "message")

		require.NoError(t, NewObserver(d).Decode(TranscoderContext{State: StateHealthy}, m))
		require.Len(t, d.delivered, 1)
	})

	t.Run("should ignore messages replayed while recovering", func(t *testing.T) {
		d := &flakyDispatcher{}
		m := agent.LogInfo(agent.NewPeer("local"), "message")

		require.NoError(t, NewObserver(d).Decode(TranscoderContext{State: StateRecovering}, m))
		require.Zero(t, d.attempts)
	})
}
