package quorum

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/pkg/errors"
)

// interface for retrieving a raft.FSM
type waler interface {
	WAL() raft.FSM
}

// CommandToMessage ...
func commandToMessage(cmd []byte) (m agent.Message, err error) {
	return m, errors.WithStack(proto.Unmarshal(cmd, &m))
}

// NewWAL ...
func NewWAL(obs chan agent.Message) WAL {
	return WAL{
		m:        &sync.RWMutex{},
		observer: obs,
	}
}

type innerstate struct {
	logs                 []agent.Message
	deploying            int32                // is a deploy process in progress.
	runningDeploy        *agent.DeployCommand // currently active deployment.
	lastSuccessfulDeploy *agent.DeployCommand // used for bootstrapping and recovering when a deploy proxy fails.
}

// WAL for the quorum.
type WAL struct {
	innerstate
	observer chan agent.Message
	m        *sync.RWMutex
}

// Apply log is invoked once a log entry is committed.
// It returns a value which will be made available in the
// ApplyFuture returned by Raft.Apply method if that
// method was called on the same Raft node as the FSM.
func (t *WAL) Apply(l *raft.Log) interface{} {
	switch l.Type {
	case raft.LogBarrier:
		log.Println("barrier invoked", l.Index, l.Term)
	case raft.LogCommand:
		if err := t.decode(l.Data); err != nil {
			return err
		}
	case raft.LogNoop:
		log.Println("noop invoked", l.Index, l.Term)
	}

	return nil
}

func (t *WAL) decode(buf []byte) error {
	var (
		err error
		m   agent.Message
	)

	if m, err = commandToMessage(buf); err != nil {
		return err
	}

	switch event := m.GetEvent().(type) {
	case *agent.Message_DeployCommand:
		if err = t.deployCommand(event.DeployCommand); err != nil {
			return err
		}
	default:
	}

	t.m.Lock()
	t.logs = append(t.logs, m)
	t.m.Unlock()

	// TODO consider moving observer into the state machine, would resolve needing
	// to mark messages as replays when restoring state.
	// ignore replayed messages.
	if !m.Replay && t.observer != nil {
		t.observer <- m
	}

	return nil
}

func (t *WAL) deployCommand(dc *agent.DeployCommand) error {
	debugx.Println("deploy command received", dc.Command.String())
	defer debugx.Println("deploy command processed", dc.Command.String())

	switch dc.Command {
	case agent.DeployCommand_Begin:
		if !atomic.CompareAndSwapInt32(&t.deploying, none, deploying) {
			return errors.New(fmt.Sprint("deploy already in progress"))
		}
		t.m.Lock()
		t.runningDeploy = dc
		t.m.Unlock()
	case agent.DeployCommand_Done:
		atomic.SwapInt32(&t.deploying, none)
		t.m.Lock()
		t.lastSuccessfulDeploy = dc
		t.runningDeploy = nil
		t.m.Unlock()
	default:
		atomic.SwapInt32(&t.deploying, none)
	}

	return nil
}

func (t *WAL) getInfo() agent.InfoResponse {
	t.m.RLock()
	defer t.m.RUnlock()

	m := agent.InfoResponse_None
	if atomic.LoadInt32(&t.deploying) == deploying {
		m = agent.InfoResponse_Deploying
	}

	return agent.InfoResponse{
		Mode:      m,
		Deploying: t.runningDeploy,
		Deployed:  t.lastSuccessfulDeploy,
	}
}

func (t *WAL) getLastSuccessfulDeploy() *agent.DeployCommand {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.lastSuccessfulDeploy
}

func (t *WAL) getRunningDeploy() *agent.DeployCommand {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.runningDeploy
}

// Snapshot is used to support log compaction. This call should
// return an FSMSnapshot which can be used to save a point-in-time
// snapshot of the FSM. Apply and Snapshot are not called in multiple
// threads, but Apply will be called concurrently with Persist. This means
// the FSM should be implemented in a fashion that allows for concurrent
// updates while a snapshot is happening.
func (t *WAL) Snapshot() (raft.FSMSnapshot, error) {
	return &walSnapshot{wal: t, max: len(t.logs)}, nil
}

// Restore is used to restore an FSM from a snapshot. It is not called
// concurrently with any other command. The FSM must discard all previous
// state.
func (t *WAL) Restore(r io.ReadCloser) (err error) {
	var (
		encoded []byte
		decoded agent.WAL
	)

	log.Println("WAL restoring")
	defer log.Println("WAL restored")

	if encoded, err = ioutil.ReadAll(r); err != nil {
		return errors.WithStack(err)
	}

	if err = proto.Unmarshal(encoded, &decoded); err != nil {
		return errors.WithStack(err)
	}

	// reset the internal state of the write ahead log.
	t.innerstate = innerstate{
		logs:      make([]agent.Message, 0, len(decoded.Messages)),
		deploying: none,
	}

	for _, m := range decoded.Messages {
		if encoded, err = proto.Marshal(m); err != nil {
			return errors.WithStack(err)
		}

		if err = t.decode(encoded); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (t *WAL) advance(n int) {
	t.m.Lock()
	t.logs = t.logs[n:]
	t.m.Unlock()
}

type walSnapshot struct {
	wal      *WAL
	min, max int
}

// Persist should dump all necessary state to the WriteCloser 'sink',
// and call sink.Close() when finished or call sink.Cancel() on error.
func (t *walSnapshot) Persist(sink raft.SnapshotSink) (err error) {
	var (
		encoded []byte
		msg     agent.Message
		state   agent.WAL
		i       int
	)

	log.Println("persist invoked")
	defer log.Println("persist completed")
	for i, msg = range t.wal.logs[:t.max] {
		switch msg.GetType() {
		// whenever we encounter a deploy command event reset the state,
		// this ensures we only keep the state of the latest deploy when compacting the fsm.
		case agent.Message_DeployCommandEvent:
			t.min = i
			state.Messages = state.Messages[:0]
		}

		tmp := msg
		tmp.Replay = true
		state.Messages = append(state.Messages, &tmp)
	}

	if encoded, err = proto.Marshal(&state); err != nil {
		sink.Cancel()
		return errors.WithStack(err)
	}

	if _, err = sink.Write(encoded); err != nil {
		sink.Cancel()
		return errors.WithStack(err)
	}

	return sink.Close()
}

// Release is invoked when we are finished with the snapshot.
func (t walSnapshot) Release() {
	t.wal.advance(t.min)
}
