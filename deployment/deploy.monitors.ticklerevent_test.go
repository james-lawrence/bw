package deployment

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/iterx"
	"github.com/stretchr/testify/require"
)

// messages is a sequence yielding the messages in order.
func messages(ms ...*agent.Message) iterx.Seq[*agent.Message] {
	return iterx.New(func(ctx context.Context, yield func(*agent.Message) bool) error {
		for _, m := range ms {
			if !yield(m) {
				return nil
			}
		}

		return nil
	})
}

// events pumps the sequence into a closed channel.
func events(t *testing.T, seq iterx.Seq[*agent.Message]) <-chan *agent.Message {
	out := make(chan *agent.Message, 16)
	for m := range seq.Each(t.Context()) {
		out <- m
	}
	require.NoError(t, seq.Err())
	close(out)

	return out
}

// waiter blocks on the condition, the returned channel is closed once the condition is signalled.
func waiter(tickle *sync.Cond) <-chan struct{} {
	var (
		ready    = make(chan struct{})
		signaled = make(chan struct{})
	)

	go func() {
		tickle.L.Lock()
		close(ready)
		tickle.Wait()
		tickle.L.Unlock()
		close(signaled)
	}()

	<-ready
	// the lock is only available once the waiter has begun waiting.
	tickle.L.Lock()
	tickle.L.Unlock()

	return signaled
}

func TestTickleOnEvents(t *testing.T) {
	var (
		peer = agent.NewPeer("node1")
	)

	for name, tc := range map[string]struct {
		message  *agent.Message
		signaled bool
		departed bool
	}{
		"completed deploys trigger a healthcheck": {
			message:  agent.DeployEvent(peer, &agent.Deploy{Stage: agent.Deploy_Completed}),
			signaled: true,
		},
		"failed deploys trigger a healthcheck": {
			message:  agent.DeployEvent(peer, &agent.Deploy{Stage: agent.Deploy_Failed}),
			signaled: true,
		},
		"deploys in progress are ignored": {
			message: agent.DeployEvent(peer, &agent.Deploy{Stage: agent.Deploy_Deploying}),
		},
		"departed nodes are reported": {
			message:  agent.NodeEvent(peer, agent.Message_Departed),
			departed: true,
		},
		"joined nodes are ignored": {
			message: agent.NodeEvent(peer, agent.Message_Joined),
		},
		"other messages are ignored": {
			message: agent.LogInfo(peer, "ignored"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			var (
				tickle   = sync.NewCond(&sync.Mutex{})
				departed = make(chan *agent.Peer, 1)
				signaled = waiter(tickle)
			)

			tickleOnEvents(t.Context(), events(t, messages(tc.message)), tickle, departed)

			select {
			case <-signaled:
				require.True(t, tc.signaled, "unexpected healthcheck")
			case <-time.After(50 * time.Millisecond):
				require.False(t, tc.signaled, "expected a healthcheck")
			}

			select {
			case p := <-departed:
				require.True(t, tc.departed, "unexpected departure")
				require.Equal(t, peer.Name, p.Name)
			default:
				require.False(t, tc.departed, "expected a departure")
			}
		})
	}

	t.Run("processes every message in the sequence", func(t *testing.T) {
		var (
			a        = agent.NewPeer("node1")
			b        = agent.NewPeer("node2")
			tickle   = sync.NewCond(&sync.Mutex{})
			departed = make(chan *agent.Peer, 2)
		)

		tickleOnEvents(t.Context(), events(t, messages(
			agent.NodeEvent(a, agent.Message_Departed),
			agent.LogInfo(a, "ignored"),
			agent.NodeEvent(b, agent.Message_Departed),
		)), tickle, departed)

		require.Equal(t, a.Name, (<-departed).Name)
		require.Equal(t, b.Name, (<-departed).Name)
	})

	t.Run("stops when the context is done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			tickleOnEvents(ctx, make(chan *agent.Message), sync.NewCond(&sync.Mutex{}), make(chan *agent.Peer))
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("expected the tickler to stop")
		}
	})
}
