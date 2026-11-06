package quorum

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

// controlled is a dispatcher with the controls of a journal, recording how they are used.
type controlled struct {
	m        sync.Mutex
	messages []*agent.Message
	retained []time.Time
	resets   int
}

func (t *controlled) Dispatch(_ context.Context, m ...*agent.Message) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.messages = append(t.messages, m...)
	return nil
}

func (t *controlled) Retain(since time.Time) {
	t.m.Lock()
	defer t.m.Unlock()
	t.retained = append(t.retained, since)
}

func (t *controlled) Reset() error {
	t.m.Lock()
	defer t.m.Unlock()
	t.resets++
	t.messages = nil
	return nil
}

// at stamps the message as created at the time.
func at(m *agent.Message, ts time.Time) *agent.Message {
	m.Id = uuid.Must(uuid.NewV7AtTime(ts)).String()
	m.Ts = ts.UnixNano()
	return m
}

func TestObserverJournal(t *testing.T) {
	var (
		peer  = agent.NewPeer("node")
		epoch = time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC)
	)

	// the archive of a deploy created at created (unix seconds, 0 for none), the deploy messages are created at ts.
	archive := func(created int64) *agent.Archive {
		return &agent.Archive{DeploymentID: []byte("d1"), Ts: created}
	}
	begin := func(created int64, ts time.Time) *agent.Message {
		return at(agent.NewDeployCommand(peer, agent.DeployCommandBegin("tester", archive(created), &agent.DeployOptions{})), ts)
	}
	done := func(created int64, ts time.Time) *agent.Message {
		return at(agent.NewDeployCommand(peer, agent.DeployCommandDone("tester", archive(created).DeployOption)), ts)
	}

	decode := func(t *testing.T, o Observer, state int64, ms ...*agent.Message) {
		t.Helper()
		for _, m := range ms {
			require.NoError(t, o.Decode(TranscoderContext{State: state}, m))
		}
	}

	t.Run("records every message", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		ms := []*agent.Message{agent.LogInfo(peer, "a"), agent.NewDeployHeartbeat(peer)}
		decode(t, o, StateHealthy, ms...)

		require.Equal(t, ids(ms), ids(c.messages))
		require.Empty(t, c.retained)
		require.Zero(t, c.resets)
	})

	t.Run("records the messages replayed by a restore", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		ms := []*agent.Message{agent.LogInfo(peer, "a"), agent.LogInfo(peer, "b")}
		decode(t, o, StateRecovering, ms...)

		require.Equal(t, ids(ms), ids(c.messages))
	})

	t.Run("a restore starts over", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		decode(t, o, StateHealthy, agent.LogInfo(peer, "before"))

		replayed := agent.LogInfo(peer, "replayed")
		decode(t, o, StateRecovering, agent.NewRestoreInitiated(peer), replayed, agent.NewRestoreCompleted(peer))

		require.Equal(t, 1, c.resets)
		require.Equal(t, []string{replayed.Id}, ids(c.messages), "the markers are local, they are not recorded")
	})

	t.Run("a completed deploy retains from when its archive was created", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		created := epoch.Add(-time.Hour)
		decode(t, o, StateHealthy, begin(created.Unix(), epoch), agent.LogInfo(peer, "working"), done(created.Unix(), epoch.Add(time.Hour)))

		require.Equal(t, millis(created), millis(c.retained...))
	})

	t.Run("each completed deploy retains from its own archive", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		first, second := epoch.Add(-3*time.Hour), epoch.Add(-time.Hour)
		decode(t, o, StateHealthy,
			begin(first.Unix(), epoch), done(first.Unix(), epoch.Add(time.Hour)),
			begin(second.Unix(), epoch.Add(2*time.Hour)), done(second.Unix(), epoch.Add(3*time.Hour)),
		)

		require.Equal(t, millis(first, second), millis(c.retained...))
	})

	t.Run("an archive without a timestamp retains from when the deploy completed", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		decode(t, o, StateHealthy, done(0, epoch.Add(time.Hour)))

		require.Equal(t, millis(epoch.Add(time.Hour)), millis(c.retained...))
	})

	t.Run("a deploy that hasnt completed retains nothing new", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		created := epoch.Add(-time.Hour)
		decode(t, o, StateHealthy,
			begin(created.Unix(), epoch), done(created.Unix(), epoch.Add(time.Hour)),
			begin(created.Unix(), epoch.Add(2*time.Hour)),
			at(agent.NewDeployCommand(peer, agent.DeployCommandFailed("tester")), epoch.Add(3*time.Hour)),
			at(agent.NewDeployCommand(peer, agent.DeployCommandRestart(nil, nil)), epoch.Add(4*time.Hour)),
		)

		require.Equal(t, millis(created), millis(c.retained...))
	})

	t.Run("a snapshot that starts at the completion retains from the archive", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		created := epoch.Add(-time.Hour)
		decode(t, o, StateRecovering, agent.NewRestoreInitiated(peer), done(created.Unix(), epoch.Add(time.Hour)))

		require.Equal(t, 1, c.resets)
		require.Equal(t, millis(created), millis(c.retained...))
	})

	t.Run("other messages never retain", func(t *testing.T) {
		c := &controlled{}
		o := NewObserver(c)
		decode(t, o, StateHealthy,
			agent.LogInfo(peer, "a"),
			agent.NodeEvent(peer, agent.Message_Joined),
			agent.DeployEvent(peer, &agent.Deploy{Stage: agent.Deploy_Completed, Archive: archive(1)}),
		)

		require.Empty(t, c.retained)
	})

	t.Run("works with a dispatcher that has no journal controls", func(t *testing.T) {
		o := NewObserver(dispatchOnly{&controlled{}})

		decode(t, o, StateHealthy, begin(1, epoch), done(1, epoch.Add(time.Hour)))
		decode(t, o, StateRecovering, agent.NewRestoreInitiated(peer))
	})
}

// dispatchOnly hides everything but dispatching.
type dispatchOnly struct{ inner agent.Dispatcher }

func (t dispatchOnly) Dispatch(ctx context.Context, m ...*agent.Message) error {
	return t.inner.Dispatch(ctx, m...)
}

// millis the instants, times with different locations are not equal even when they are the same moment.
func millis(ts ...time.Time) (r []int64) {
	for _, ts := range ts {
		r = append(r, ts.UnixMilli())
	}
	return r
}
