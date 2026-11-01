package deployment

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/stretchr/testify/require"
)

func stage(s agent.Deploy_Stage) OperationFunc {
	return func(context.Context, *agent.Peer) (*agent.Deploy, error) {
		return &agent.Deploy{Stage: s}, nil
	}
}

// await runs the monitor with a fast periodic tickler, returning the pending nodes' done channels.
func await(ctx context.Context, q chan *pending, expected int, check operation, additional ...MonitorTickler) error {
	return NewMonitor(MonitorTicklerPeriodic(5*time.Millisecond)).Await(ctx, q, expected, agentutil.DiscardDispatcher{}, nil, check, additional...)
}

func TestMonitorAwait(t *testing.T) {
	t.Run("returns immediately when nothing is expected", func(t *testing.T) {
		require.NoError(t, await(t.Context(), make(chan *pending), 0, stage(agent.Deploy_Completed)))
	})

	t.Run("completes once every expected node reports without the queue closing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		tasks := []*pending{
			newPending(agent.NewPeer("node1"), time.Second),
			newPending(agent.NewPeer("node2"), time.Second),
		}

		q := make(chan *pending, len(tasks))
		for _, task := range tasks {
			q <- task
		}

		require.NoError(t, await(ctx, q, len(tasks), stage(agent.Deploy_Completed)))
		for _, task := range tasks {
			require.NoError(t, <-task.done)
		}
	})

	t.Run("reports that don't need monitoring count towards completion", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		var checks atomic.Int64
		check := OperationFunc(func(context.Context, *agent.Peer) (*agent.Deploy, error) {
			checks.Add(1)
			return &agent.Deploy{Stage: agent.Deploy_Completed}, nil
		})

		task := newPending(agent.NewPeer("node1"), time.Second)
		q := make(chan *pending, 3)
		q <- nil
		q <- task
		q <- nil

		require.NoError(t, await(ctx, q, 3, check))
		require.EqualValues(t, 1, checks.Load())
		require.NoError(t, <-task.done)
	})

	t.Run("waits for the node to complete", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		var checks atomic.Int64
		check := OperationFunc(func(context.Context, *agent.Peer) (*agent.Deploy, error) {
			if checks.Add(1) < 3 {
				return &agent.Deploy{Stage: agent.Deploy_Deploying}, nil
			}

			return &agent.Deploy{Stage: agent.Deploy_Completed}, nil
		})

		task := newPending(agent.NewPeer("node1"), time.Second)
		q := make(chan *pending, 1)
		q <- task

		require.NoError(t, await(ctx, q, 1, check))
		require.GreaterOrEqual(t, checks.Load(), int64(3))
		require.NoError(t, <-task.done)
	})

	t.Run("retries when the healthcheck errors", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		var checks atomic.Int64
		check := OperationFunc(func(context.Context, *agent.Peer) (*agent.Deploy, error) {
			if checks.Add(1) < 3 {
				return nil, errors.New("unavailable")
			}

			return &agent.Deploy{Stage: agent.Deploy_Completed}, nil
		})

		task := newPending(agent.NewPeer("node1"), time.Second)
		q := make(chan *pending, 1)
		q <- task

		require.NoError(t, await(ctx, q, 1, check))
		require.GreaterOrEqual(t, checks.Load(), int64(3))
	})

	t.Run("reports a failed deployment to the node", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		task := newPending(agent.NewPeer("node1"), time.Second)
		q := make(chan *pending, 1)
		q <- task

		require.NoError(t, await(ctx, q, 1, stage(agent.Deploy_Failed)))
		require.ErrorContains(t, <-task.done, "deployment has failed")
	})

	t.Run("returns the context error when the nodes never all report", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		q := make(chan *pending, 1)
		q <- newPending(agent.NewPeer("node1"), time.Second)

		require.ErrorIs(t, await(ctx, q, 2, stage(agent.Deploy_Completed)), context.DeadlineExceeded)
	})

	t.Run("returns the context error when a node never completes", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		q := make(chan *pending, 1)
		q <- newPending(agent.NewPeer("node1"), time.Second)

		require.ErrorIs(t, await(ctx, q, 1, stage(agent.Deploy_Deploying)), context.DeadlineExceeded)
	})

	t.Run("stops waiting for nodes that departed the cluster", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		peer := agent.NewPeer("node1")
		task := newPending(peer, time.Second)
		q := make(chan *pending, 1)
		q <- task

		var once sync.Once
		departing := func(ctx context.Context, tickle *sync.Cond, departed chan *agent.Peer) {
			once.Do(func() {
				select {
				case departed <- peer:
				case <-ctx.Done():
				}
			})
		}

		require.NoError(t, await(ctx, q, 1, stage(agent.Deploy_Deploying), departing))

		select {
		case _, open := <-task.done:
			require.False(t, open, "expected the departed node's task to be released")
		default:
			t.Fatal("expected the departed node's task to be released")
		}
	})
}
