package quorum

import (
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/x/debugx"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

const (
	none int32 = iota
	deploying
)

// NewStateMachine ...
func NewStateMachine() *StateMachine {
	return &StateMachine{
		m:        &sync.RWMutex{},
		EventBus: agent.NewEventBus(),
	}
}

// MessageToCommand ...
func MessageToCommand(m agent.Message) ([]byte, error) {
	return proto.Marshal(&m)
}

// CommandToMessage ...
func CommandToMessage(cmd []byte) (m agent.Message, err error) {
	return m, errors.WithStack(proto.Unmarshal(cmd, &m))
}

// StateMachine ...
type StateMachine struct {
	agent.EventBus
	m         *sync.RWMutex
	details   agent.StatusResponse
	deploying int32
}

// Apply log is invoked once a log entry is committed.
// It returns a value which will be made available in the
// ApplyFuture returned by Raft.Apply method if that
// method was called on the same Raft node as the FSM.
func (t *StateMachine) Apply(l *raft.Log) interface{} {
	switch l.Type {
	case raft.LogBarrier:
		log.Println("barrier invoked", l.Index, l.Term)
	case raft.LogCommand:
		debugx.Println("command invoked", l.Index, l.Term)
		return t.decode(l.Data)
	case raft.LogNoop:
		log.Println("noop invoked", l.Index, l.Term)
	}

	return nil
}

func (t *StateMachine) deployCommand(dc *agent.DeployCommand) error {
	switch dc.Command {
	case agent.DeployCommand_Begin:
		if !atomic.CompareAndSwapInt32(&t.deploying, none, deploying) {
			return errors.New(fmt.Sprint("deploy already in progress"))
		}
	default:
		atomic.SwapInt32(&t.deploying, none)
	}

	return nil
}

func (t *StateMachine) decode(buf []byte) error {
	var (
		err error
		m   agent.Message
	)

	if m, err = CommandToMessage(buf); err != nil {
		return err
	}

	switch event := m.GetEvent().(type) {
	case *agent.Message_DeployCommand:
		if err = t.deployCommand(event.DeployCommand); err != nil {
			return err
		}
	default:
	}

	debugx.Println("dispatching into event bus")
	t.EventBus.Dispatch(m)
	debugx.Println("dispatched into event bus")

	return nil
}

// Snapshot is used to support log compaction. This call should
// return an FSMSnapshot which can be used to save a point-in-time
// snapshot of the FSM. Apply and Snapshot are not called in multiple
// threads, but Apply will be called concurrently with Persist. This means
// the FSM should be implemented in a fashion that allows for concurrent
// updates while a snapshot is happening.
func (t *StateMachine) Snapshot() (raft.FSMSnapshot, error) {
	return quorumFSMSnapshot{details: t.details}, nil
}

// Restore is used to restore an FSM from a snapshot. It is not called
// concurrently with any other command. The FSM must discard all previous
// state.
func (t *StateMachine) Restore(r io.ReadCloser) error {
	t.details = agent.StatusResponse{}
	return nil
}

// Details includes information about the details of the quorum.
// who its members are, the latest deploys.
func (t StateMachine) Details() (d agent.StatusResponse, err error) {
	return t.details, nil
}

type quorumFSMSnapshot struct {
	details agent.StatusResponse
}

// Persist should dump all necessary state to the WriteCloser 'sink',
// and call sink.Close() when finished or call sink.Cancel() on error.
func (t quorumFSMSnapshot) Persist(sink raft.SnapshotSink) (err error) {
	var (
		state []byte
	)

	if state, err = proto.Marshal(&t.details); err != nil {
		sink.Cancel()
		return errors.WithStack(err)
	}

	if _, err = sink.Write(state); err != nil {
		sink.Cancel()
		return errors.WithStack(err)
	}

	return sink.Close()
}

// Release is invoked when we are finished with the snapshot.
func (t quorumFSMSnapshot) Release() {

}
