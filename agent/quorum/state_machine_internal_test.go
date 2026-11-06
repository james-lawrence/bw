package quorum

import (
	"context"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"

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

	// a restore resets the journal, see TestObserverJournal, the messages it replays rebuild it.
	t.Run("should journal messages replayed while recovering", func(t *testing.T) {
		d := &flakyDispatcher{}
		m := agent.LogInfo(agent.NewPeer("local"), "message")

		require.NoError(t, NewObserver(d).Decode(TranscoderContext{State: StateRecovering}, m))
		require.Len(t, d.delivered, 1)
	})
}

// writes beyond what raft can take are told to back off promptly rather than held for their deadline.
func TestStateMachineShouldTellCallersToBackOffWhenTooManyWritesAreInFlight(t *testing.T) {
	// admission happens before raft is touched, so no cluster is required.
	sm := NewMachine(t.Context(), agent.NewPeer("node"), nil)

	// occupy every slot.
	for i := 0; i < cap(sm.inflight); i++ {
		sm.inflight <- struct{}{}
	}

	start := time.Now()
	err := sm.writeWAL(t.Context(), agent.LogInfo(agent.NewPeer("node"), "message"), 100*time.Millisecond)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Less(t, time.Since(start), time.Second)
}
