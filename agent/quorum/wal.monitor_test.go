package quorum

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"google.golang.org/protobuf/proto"
)

type failingDispatcher struct{ cause error }

func (t failingDispatcher) Dispatch(context.Context, ...*agent.Message) error {
	return t.cause
}

func TestMonitoredWAL(t *testing.T) {
	var (
		leader = func() *agent.Peer { return p }
		follow = func() *agent.Peer { return nil }
	)

	newWAL := func(d agent.Dispatcher, local func() *agent.Peer) (monitoredWAL, *applyMonitor) {
		monitor := newApplyMonitor(d, local)
		monitor.threshold = 10 * time.Millisecond
		w := NewWAL(slowTranscoder{delay: 20 * time.Millisecond})
		return newMonitoredWAL(&w, monitor), monitor
	}

	apply := func(t *testing.T, wal monitoredWAL, m *agent.Message) interface{} {
		raw, err := proto.Marshal(m)
		require.NoError(t, err)
		return wal.Apply(&raft.Log{Type: raft.LogCommand, Data: raw})
	}

	deploy := func(t *testing.T, wal monitoredWAL) {
		apply(t, wal, agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Begin}))
		apply(t, wal, agent.LogInfo(p, "a"))
		apply(t, wal, agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Done}))
	}

	t.Run("should report the first slow apply and the max duration of each deploy", func(t *testing.T) {
		d := &recordingDispatcher{}
		wal, monitor := newWAL(d, leader)

		deploy(t, wal)
		deploy(t, wal)

		require.NoError(t, monitor.Close())
		msgs := d.all()
		require.Len(t, msgs, 4)
		require.Contains(t, msgs[0].GetLog().GetLog(), "WARN - fsm apply exceeded")
		require.Contains(t, msgs[1].GetLog().GetLog(), "fsm apply max duration")
		require.Contains(t, msgs[2].GetLog().GetLog(), "WARN - fsm apply exceeded")
		require.Contains(t, msgs[3].GetLog().GetLog(), "fsm apply max duration")
	})

	t.Run("should not report when no apply is slow", func(t *testing.T) {
		d := &recordingDispatcher{}
		wal, monitor := newWAL(d, leader)
		monitor.threshold = time.Minute

		apply(t, wal, agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Begin}))
		require.NoError(t, monitor.Close())
		require.Empty(t, d.all())
	})

	t.Run("should not report when not the leader", func(t *testing.T) {
		d := &recordingDispatcher{}
		wal, monitor := newWAL(d, follow)

		deploy(t, wal)

		require.NoError(t, monitor.Close())
		require.Empty(t, d.all())
	})

	t.Run("should send the queued reports before close returns", func(t *testing.T) {
		d := &recordingDispatcher{}
		wal, monitor := newWAL(&delayedDispatcher{Dispatcher: d, delay: 50 * time.Millisecond}, leader)

		deploy(t, wal)

		require.NoError(t, monitor.Close())
		require.Len(t, d.all(), 2)
	})

	t.Run("should return the failure to report from close", func(t *testing.T) {
		cause := errors.New("boom")
		wal, monitor := newWAL(failingDispatcher{cause: cause}, leader)

		deploy(t, wal)

		require.ErrorIs(t, monitor.Close(), cause)
	})

	t.Run("should return the error of the wrapped wal", func(t *testing.T) {
		wal, monitor := newWAL(&recordingDispatcher{}, leader)
		defer monitor.Close()

		require.Error(t, apply(t, wal, agent.NewUploadProgress(p, 1, 2)).(error))
	})

	t.Run("should pass the other log types through", func(t *testing.T) {
		wal, monitor := newWAL(&recordingDispatcher{}, leader)
		defer monitor.Close()

		require.Nil(t, wal.Apply(&raft.Log{Type: raft.LogNoop}))
		require.Nil(t, wal.Apply(&raft.Log{Type: raft.LogBarrier}))
	})
}

// delayedDispatcher slows every dispatch, leaving reports queued.
type delayedDispatcher struct {
	agent.Dispatcher
	delay time.Duration
}

func (t *delayedDispatcher) Dispatch(ctx context.Context, m ...*agent.Message) error {
	time.Sleep(t.delay)
	return t.Dispatcher.Dispatch(ctx, m...)
}
