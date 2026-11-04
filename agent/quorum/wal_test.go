package quorum

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	"github.com/hashicorp/raft"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
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
var _ = DescribeTable(
	"WAL Snapshot/Restore", func(commands ...*agent.Message) {
		deployment := newDeployment(nil)
		mobs, err := journal.New(GinkgoT().TempDir())
		Expect(err).To(Succeed())
		obs := NewObserver(mobs)
		wal := NewWAL(
			NewTranscoder(
				deployment,
				obs,
			),
		)

		Expect(snapshotRestore(wal, bytes.NewBuffer(nil), commands...)).To(Succeed())
	},
	Entry(
		"successful deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
	),
	Entry(
		"failed deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandFailedQuick()),
	),
	Entry(
		"cancelled deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
	),
	Entry(
		"sequential successful deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
	),
	Entry(
		"sequential cancelled deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
	),
	Entry(
		"sequential failed, cancelled deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandFailedQuick()),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
	),
	Entry(
		"sequential deployment begin",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandRestart()),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", archive, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
	),
)

var _ = DescribeTable(
	"WAL Restore", func(commands ...proto.Message) {
		deployment := newDeployment(nil)
		mobs, err := journal.New(GinkgoT().TempDir())
		Expect(err).To(Succeed())
		obs := NewObserver(mobs)
		wal := NewWAL(
			NewTranscoder(
				deployment,
				obs,
			),
		)
		buf := bytes.NewBuffer(nil)
		Expect(EncodeEvery(buf, commands...)).To(Succeed())
		Expect(wal.Restore(io.NopCloser(buf))).To(Succeed())
	},
	Entry(
		"successful deployment",
		agent.NewWALPreamble(),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
	),
	Entry(
		"sequential deployment begin",
		agent.NewWALPreamble(),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
	),
	Entry(
		"sequential deployment begin",
		agent.NewWALPreamble(),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandRestart()),
		agent.NewDeployCommand(p, agent.DeployCommandDone("foo", archive.DeployOption)),
	),
)

var _ = Describe("WAL", func() {
	It("should reject upload progress messages", func() {
		mobs, err := journal.New(GinkgoT().TempDir())
		Expect(err).To(Succeed())
		wal := NewWAL(NewTranscoder(newDeployment(nil), NewObserver(mobs)))

		raw, err := proto.Marshal(agent.NewUploadProgress(p, 1, 2))
		Expect(err).To(Succeed())
		_, derr := wal.decode(wal.ctx, raw)
		Expect(derr).To(HaveOccurred())
	})

	Describe("slow apply", func() {
		It("should report the first slow apply and the max duration of a deploy", func() {
			d := &recordingDispatcher{}
			monitor := newApplyMonitor(d, func() *agent.Peer { return p })
			monitor.threshold = 10 * time.Millisecond
			w := NewWAL(slowTranscoder{delay: 20 * time.Millisecond})
			wal := newMonitoredWAL(&w, monitor)

			apply := func(m *agent.Message) {
				raw, err := proto.Marshal(m)
				Expect(err).To(Succeed())
				Expect(wal.Apply(&raft.Log{Type: raft.LogCommand, Data: raw})).To(BeNil())
			}

			apply(agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Begin}))
			apply(agent.LogInfo(p, "a"))
			apply(agent.LogInfo(p, "b"))
			Eventually(d.all).Should(HaveLen(1))
			Expect(d.all()[0].GetLog().GetLog()).To(ContainSubstring("WARN - fsm apply exceeded"))

			apply(agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Done}))
			Eventually(d.all).Should(HaveLen(2))
			Expect(d.all()[1].GetLog().GetLog()).To(ContainSubstring("fsm apply max duration"))
		})
	})
})

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
