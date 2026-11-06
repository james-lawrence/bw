package quorum

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/raft"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func NewInmemSnapshotSink(b io.ReadWriter) *InmemSnapshotSink {
	return &InmemSnapshotSink{
		contents: b,
	}
}

// InmemSnapshotSink implements SnapshotSink in memory
type InmemSnapshotSink struct {
	contents io.ReadWriter
}

// Write appends the given bytes to the snapshot contents
func (s *InmemSnapshotSink) Write(p []byte) (n int, err error) {
	written, err := s.contents.Write(p)
	return written, err
}

// Close updates the Size and is otherwise a no-op
func (s *InmemSnapshotSink) Close() error {
	return nil
}

// ID returns the ID of the SnapshotMeta
func (s *InmemSnapshotSink) ID() string {
	return "inmemsink"
}

// Cancel returns successfully with a nil error
func (s *InmemSnapshotSink) Cancel() error {
	return nil
}

func messagesToCommands(msg ...*agent.Message) (cmd [][]byte, err error) {
	for _, m := range msg {
		encoded, err := proto.Marshal(m)
		if err != nil {
			return cmd, err
		}
		cmd = append(cmd, encoded)
	}

	return cmd, nil
}

func snapshotRestore(w WAL, buff io.ReadWriter, commands ...*agent.Message) error {
	cmds, err := messagesToCommands(commands...)
	if err != nil {
		return err
	}

	for _, cmd := range cmds {
		if _, err := w.decode(w.ctx, cmd); err != nil {
			return err
		}
	}

	s, err := w.Snapshot()
	if err != nil {
		return err
	}
	defer s.Release()

	sink := NewInmemSnapshotSink(buff)
	if err = s.Persist(sink); err != nil {
		return err
	}

	if err = w.Restore(io.NopCloser(sink.contents)); err != nil {
		return err
	}

	return nil
}

var (
	p       = agent.NewPeer("abc123")
	archive = &agent.Archive{}
)

func TestWALSnapshotRestore(t *testing.T) {
	for _, c := range []struct {
		name     string
		commands []*agent.Message
	}{
		{
			"successful deployment",
			[]*agent.Message{
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
			},
		},
		{
			"failed deployment",
			[]*agent.Message{
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandFailedQuick()),
			},
		},
		{
			"cancelled deployment",
			[]*agent.Message{
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
			},
		},
		{
			"sequential successful deployment",
			[]*agent.Message{
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
			},
		},
		{
			"sequential cancelled deployment",
			[]*agent.Message{
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
			},
		},
		{
			"sequential failed, cancelled deployment",
			[]*agent.Message{
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandFailedQuick()),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
			},
		},
		{
			"sequential deployment begin",
			[]*agent.Message{
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandRestart(nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			deployment := newDeployment(nil)
			mobs, err := journal.New(t.TempDir())
			require.NoError(t, err)
			t.Cleanup(func() { mobs.Close() })
			obs := NewObserver(mobs)
			wal := NewWAL(
				NewTranscoder(
					deployment,
					obs,
				),
			)

			require.NoError(t, snapshotRestore(wal, bytes.NewBuffer(nil), c.commands...))
		})
	}
}

func TestWALRestore(t *testing.T) {
	for _, c := range []struct {
		name     string
		commands []proto.Message
	}{
		{
			"successful deployment",
			[]proto.Message{
				agent.NewWALPreamble(),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
			},
		},
		{
			"sequential deployment begin",
			[]proto.Message{
				agent.NewWALPreamble(),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
			},
		},
		{
			"sequential deployment begin",
			[]proto.Message{
				agent.NewWALPreamble(),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
				agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
				agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandRestart(nil, nil)),
				agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			deployment := newDeployment(nil)
			mobs, err := journal.New(t.TempDir())
			require.NoError(t, err)
			t.Cleanup(func() { mobs.Close() })
			obs := NewObserver(mobs)
			wal := NewWAL(
				NewTranscoder(
					deployment,
					obs,
				),
			)

			buf := bytes.NewBuffer(nil)
			require.NoError(t, EncodeEvery(buf, c.commands...))
			require.NoError(t, wal.Restore(io.NopCloser(buf)))
		})
	}
}

func TestWAL(t *testing.T) {
	t.Run("should reject upload progress messages", func(t *testing.T) {
		mobs, err := journal.New(t.TempDir())
		require.NoError(t, err)
		t.Cleanup(func() { mobs.Close() })
		wal := NewWAL(NewTranscoder(newDeployment(nil), NewObserver(mobs)))
		raw, err := proto.Marshal(agent.NewUploadProgress(p, 1, 2))
		require.NoError(t, err)
		_, derr := wal.decode(wal.ctx, raw)
		require.Error(t, derr)
	})

	t.Run("slow apply", func(t *testing.T) {
		t.Run("should report the first slow apply and the max duration of a deploy", func(t *testing.T) {
			d := &recordingDispatcher{}
			monitor := newApplyMonitor(d, func() *agent.Peer { return p })
			monitor.threshold = 10 * time.Millisecond
			w := NewWAL(slowTranscoder{delay: 20 * time.Millisecond})
			wal := newMonitoredWAL(&w, monitor)

			apply := func(m *agent.Message) {
				raw, err := proto.Marshal(m)
				require.NoError(t, err)
				require.Nil(t, wal.Apply(&raft.Log{Type: raft.LogCommand, Data: raw}))
			}

			apply(agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Begin}))
			apply(agent.LogInfo(p, "a"))
			apply(agent.LogInfo(p, "b"))
			require.Eventually(t, func() bool { return len(d.all()) == 1 }, time.Second, 10*time.Millisecond)
			require.Contains(t, d.all()[0].GetLog().GetLog(), "WARN - fsm apply exceeded")

			apply(agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Done}))
			require.Eventually(t, func() bool { return len(d.all()) == 2 }, time.Second, 10*time.Millisecond)
			require.Contains(t, d.all()[1].GetLog().GetLog(), "fsm apply max duration")
		})
	})
}

type slowTranscoder struct{ delay time.Duration }

func (t slowTranscoder) Decode(TranscoderContext, *agent.Message) error {
	time.Sleep(t.delay)
	return nil
}

func (t slowTranscoder) Encode(io.Writer) error { return nil }

type recordingDispatcher struct {
	m    sync.Mutex
	msgs []*agent.Message
}

func (t *recordingDispatcher) Dispatch(_ context.Context, ms ...*agent.Message) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.msgs = append(t.msgs, ms...)
	return nil
}

func (t *recordingDispatcher) all() []*agent.Message {
	t.m.Lock()
	defer t.m.Unlock()
	return append([]*agent.Message(nil), t.msgs...)
}
