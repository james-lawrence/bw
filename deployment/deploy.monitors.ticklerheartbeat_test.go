package deployment

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

// heartbeats counts the heartbeats dispatched.
type heartbeats struct {
	m sync.Mutex
	n int
}

func (t *heartbeats) Dispatch(_ context.Context, ms ...*agent.Message) error {
	t.m.Lock()
	defer t.m.Unlock()

	for _, m := range ms {
		if m.Type == agent.Message_DeployHeartbeat {
			t.n++
		}
	}

	return nil
}

func (t *heartbeats) count() int {
	t.m.Lock()
	defer t.m.Unlock()
	return t.n
}

func TestMonitorTicklerHeartbeat(t *testing.T) {
	local := agent.NewPeer("local")

	t.Run("emits heartbeats until the context is done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var (
			sent = new(heartbeats)
			done = make(chan struct{})
		)

		go func() {
			defer close(done)
			MonitorTicklerHeartbeat(local, sent, 5*time.Millisecond)(ctx, nil, nil)
		}()

		require.Eventually(t, func() bool { return sent.count() >= 3 }, 2*time.Second, time.Millisecond)

		cancel()
		<-done

		settled := sent.count()
		time.Sleep(30 * time.Millisecond)
		require.Equal(t, settled, sent.count())
	})

	t.Run("runs while the monitor awaits and stops once it completes", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		var (
			sent  = new(heartbeats)
			task  = newPending(agent.NewPeer("node1"), time.Second)
			q     = make(chan *pending, 1)
			calls atomic.Int64
		)
		q <- task

		// the node only completes on the fifth check, giving the heartbeat time to run.
		check := OperationFunc(func(context.Context, *agent.Peer) (*agent.Deploy, error) {
			if calls.Add(1) < 5 {
				return &agent.Deploy{Stage: agent.Deploy_Deploying}, nil
			}

			return &agent.Deploy{Stage: agent.Deploy_Completed}, nil
		})

		require.NoError(t, NewMonitor(MonitorTicklerPeriodic(20*time.Millisecond)).Await(
			ctx, q, 1, sent, nil, check, MonitorTicklerHeartbeat(local, sent, 5*time.Millisecond),
		))
		require.GreaterOrEqual(t, sent.count(), 3)

		settled := sent.count()
		time.Sleep(30 * time.Millisecond)
		require.Equal(t, settled, sent.count())
	})
}
