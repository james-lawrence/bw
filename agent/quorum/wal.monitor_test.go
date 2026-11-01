package quorum

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
)

type failingDispatcher struct{ cause error }

func (t failingDispatcher) Dispatch(context.Context, ...*agent.Message) error {
	return t.cause
}

var _ = Describe("monitoredWAL", func() {
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

	apply := func(wal monitoredWAL, m *agent.Message) interface{} {
		raw, err := proto.Marshal(m)
		Expect(err).To(Succeed())
		return wal.Apply(&raft.Log{Type: raft.LogCommand, Data: raw})
	}

	deploy := func(wal monitoredWAL) {
		apply(wal, agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Begin}))
		apply(wal, agent.LogEvent(p, "a"))
		apply(wal, agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Done}))
	}

	It("should report the first slow apply and the max duration of each deploy", func() {
		d := &recordingDispatcher{}
		wal, monitor := newWAL(d, leader)

		deploy(wal)
		deploy(wal)

		Expect(monitor.Close()).To(Succeed())
		msgs := d.all()
		Expect(msgs).To(HaveLen(4))
		Expect(msgs[0].GetLog().GetLog()).To(ContainSubstring("WARN - fsm apply exceeded"))
		Expect(msgs[1].GetLog().GetLog()).To(ContainSubstring("fsm apply max duration"))
		Expect(msgs[2].GetLog().GetLog()).To(ContainSubstring("WARN - fsm apply exceeded"))
		Expect(msgs[3].GetLog().GetLog()).To(ContainSubstring("fsm apply max duration"))
	})

	It("should not report when no apply is slow", func() {
		d := &recordingDispatcher{}
		wal, monitor := newWAL(d, leader)
		monitor.threshold = time.Minute

		apply(wal, agent.NewDeployCommand(p, &agent.DeployCommand{Command: agent.DeployCommand_Begin}))
		Expect(monitor.Close()).To(Succeed())
		Expect(d.all()).To(BeEmpty())
	})

	It("should not report when not the leader", func() {
		d := &recordingDispatcher{}
		wal, monitor := newWAL(d, follow)

		deploy(wal)

		Expect(monitor.Close()).To(Succeed())
		Expect(d.all()).To(BeEmpty())
	})

	It("should send the queued reports before close returns", func() {
		d := &recordingDispatcher{}
		wal, monitor := newWAL(&delayedDispatcher{Dispatcher: d, delay: 50 * time.Millisecond}, leader)

		deploy(wal)

		Expect(monitor.Close()).To(Succeed())
		Expect(d.all()).To(HaveLen(2))
	})

	It("should return the failure to report from close", func() {
		cause := errors.New("boom")
		wal, monitor := newWAL(failingDispatcher{cause: cause}, leader)

		deploy(wal)

		Expect(monitor.Close()).To(MatchError(cause))
	})

	It("should return the error of the wrapped wal", func() {
		wal, monitor := newWAL(&recordingDispatcher{}, leader)
		defer monitor.Close()

		Expect(apply(wal, agent.NewUploadProgress(p, 1, 2))).To(HaveOccurred())
	})

	It("should pass the other log types through", func() {
		wal, monitor := newWAL(&recordingDispatcher{}, leader)
		defer monitor.Close()

		Expect(wal.Apply(&raft.Log{Type: raft.LogNoop})).To(BeNil())
		Expect(wal.Apply(&raft.Log{Type: raft.LogBarrier})).To(BeNil())
	})
})

// delayedDispatcher slows every dispatch, leaving reports queued.
type delayedDispatcher struct {
	agent.Dispatcher
	delay time.Duration
}

func (t *delayedDispatcher) Dispatch(ctx context.Context, m ...*agent.Message) error {
	time.Sleep(t.delay)
	return t.Dispatcher.Dispatch(ctx, m...)
}
