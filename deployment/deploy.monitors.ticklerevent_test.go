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

// ignoreCommands for tests not interested in the deploy commands.
func ignoreCommands(*agent.DeployCommand) {}

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

			tickleOnEvents(t.Context(), events(t, messages(tc.message)), tickle, departed, ignoreCommands)

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
		)), tickle, departed, ignoreCommands)

		require.Equal(t, a.Name, (<-departed).Name)
		require.Equal(t, b.Name, (<-departed).Name)
	})

	t.Run("stops when the context is done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			tickleOnEvents(ctx, make(chan *agent.Message), sync.NewCond(&sync.Mutex{}), make(chan *agent.Peer), ignoreCommands)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("expected the tickler to stop")
		}
	})
}

func TestSupersedes(t *testing.T) {
	var (
		local    = agent.NewPeer("coordinator")
		other    = agent.NewPeer("other")
		id       = []byte("deployment")
		other_id = []byte("another deployment")
	)

	archive := func(id []byte) *agent.Archive { return &agent.Archive{DeploymentID: id} }

	for name, tc := range map[string]struct {
		id      []byte
		command *agent.DeployCommand
		stopped bool
	}{
		"restart naming this node for the deployment":     {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Restart, Archive: archive(id), Previous: local}, stopped: true},
		"restart naming another node":                     {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Restart, Archive: archive(id), Previous: other}},
		"restart without a previous coordinator":          {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Restart, Archive: archive(id)}},
		"restart naming this node for another deployment": {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Restart, Archive: archive(other_id), Previous: local}},
		"restart when the deployment is unknown":          {command: &agent.DeployCommand{Command: agent.DeployCommand_Restart, Previous: local}},
		"cancel":                                          {id: id, command: agent.DeployCommandCancel("someone"), stopped: true},
		"cancel when the deployment is unknown":           {command: agent.DeployCommandCancel("someone"), stopped: true},
		"done for the deployment":                         {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: archive(id)}, stopped: true},
		"failed for the deployment":                       {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: archive(id)}, stopped: true},
		"done for another deployment":                     {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: archive(other_id)}},
		"done without an archive":                         {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Done}},
		"done when the deployment is unknown":             {command: &agent.DeployCommand{Command: agent.DeployCommand_Done}},
		"begin":                                           {id: id, command: &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: archive(id)}},
	} {
		t.Run(name, func(t *testing.T) {
			err := supersedes(local, tc.id)(tc.command)
			require.Equal(t, tc.stopped, err != nil, "%v", err)
		})
	}
}

func TestTickleOnEventsStopsTheDeploy(t *testing.T) {
	var (
		local = agent.NewPeer("coordinator")
		id    = []byte("deployment")
		other = agent.NewPeer("other")
	)

	run := func(t *testing.T, ms ...*agent.Message) (stops []error) {
		tickleOnEvents(
			t.Context(),
			events(t, messages(ms...)),
			sync.NewCond(&sync.Mutex{}),
			make(chan *agent.Peer, 1),
			func(dc *agent.DeployCommand) {
				if cause := supersedes(local, id)(dc); cause != nil {
					stops = append(stops, cause)
				}
			},
		)
		return stops
	}

	t.Run("a restart naming this coordinator stops it", func(t *testing.T) {
		restart := agent.DeployCommandRestart(local, &agent.DeployCommand{Archive: &agent.Archive{DeploymentID: id}})
		require.Len(t, run(t, agent.LogInfo(other, "unrelated"), agent.NewDeployCommand(other, restart)), 1)
	})

	t.Run("the restart doesnt stop the coordinator that replaced it", func(t *testing.T) {
		restart := agent.DeployCommandRestart(other, &agent.DeployCommand{Archive: &agent.Archive{DeploymentID: id}})
		require.Empty(t, run(t, agent.NewDeployCommand(local, restart)))
	})

	t.Run("unrelated events dont stop it", func(t *testing.T) {
		require.Empty(t, run(t, agent.LogInfo(other, "hello"), agent.NodeEvent(other, agent.Message_Joined)))
	})
}
