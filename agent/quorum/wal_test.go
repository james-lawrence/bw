package quorum

import (
	"bytes"
	"io"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/observers"

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
		if err := w.decode(w.ctx, cmd); err != nil {
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

var p = agent.NewPeer("abc123")
var _ = DescribeTable(
	"WAL Snapshot/Restore", func(commands ...*agent.Message) {
		deployment := newDeployment(nil, nil)
		mobs, err := observers.NewMemory()
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
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone()),
	),
	Entry(
		"failed deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandFailedQuick()),
	),
	Entry(
		"cancelled deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
	),
	Entry(
		"sequential successful deployment",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone()),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone()),
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
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandFailedQuick()),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone()),
	),
	Entry(
		"sequential deployment begin",
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandRestart()),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone()),
	),
)

var _ = DescribeTable(
	"WAL Restore", func(commands ...proto.Message) {
		deployment := newDeployment(nil, nil)
		mobs, err := observers.NewMemory()
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
		agent.NewDeployCommand(p, agent.DeployCommandDone()),
	),
	Entry(
		"sequential deployment begin",
		agent.NewWALPreamble(),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandDone()),
	),
	Entry(
		"sequential deployment begin",
		agent.NewWALPreamble(),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
		agent.NewDeployCommand(p, agent.DeployCommandCancel("bar")),
		agent.NewDeployCommand(p, agent.DeployCommandBegin("foo", nil, nil)),
		agent.NewDeployCommand(p, agent.DeployCommandRestart()),
		agent.NewDeployCommand(p, agent.DeployCommandDone()),
	),
)
