package agent

import (
	"io"
	"log"
	"sync"

	"bitbucket.org/jatone/bearded-wookie/x/debugx"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type cluster interface{}

// NewQuorum ...
func NewQuorum() *Quorum {
	return &Quorum{
		m:        &sync.RWMutex{},
		EventBus: NewEventBus(),
	}
}

// MessageToCommand ...
func MessageToCommand(m Message) ([]byte, error) {
	return proto.Marshal(&m)
}

// CommandToMessage ...
func CommandToMessage(cmd []byte) (m Message, err error) {
	return m, errors.WithStack(proto.Unmarshal(cmd, &m))
}

// Quorum ...
type Quorum struct {
	EventBus
	m       *sync.RWMutex
	details Details
}

// Apply log is invoked once a log entry is committed.
// It returns a value which will be made available in the
// ApplyFuture returned by Raft.Apply method if that
// method was called on the same Raft node as the FSM.
func (t *Quorum) Apply(l *raft.Log) interface{} {
	switch l.Type {
	case raft.LogAddPeer:
		log.Println("insert peer invoked", l.Index, l.Term)
	case raft.LogRemovePeer:
		log.Println("remove peer invoked", l.Index, l.Term)
	case raft.LogBarrier:
		log.Println("barrier invoked", l.Index, l.Term)
	case raft.LogCommand:
		debugx.Println("command invoked", l.Index, l.Term)
		t.decode(l.Data)
	case raft.LogNoop:
		log.Println("noop invoked", l.Index, l.Term)
	}

	return nil
}

func (t *Quorum) decode(buf []byte) {
	var (
		err error
		m   Message
	)

	if m, err = CommandToMessage(buf); err != nil {
		log.Println("failed to decode command", err)
		return
	}

	t.EventBus.Dispatch(m)
}

// Snapshot is used to support log compaction. This call should
// return an FSMSnapshot which can be used to save a point-in-time
// snapshot of the FSM. Apply and Snapshot are not called in multiple
// threads, but Apply will be called concurrently with Persist. This means
// the FSM should be implemented in a fashion that allows for concurrent
// updates while a snapshot is happening.
func (t *Quorum) Snapshot() (raft.FSMSnapshot, error) {
	return quorumSnapshot{details: t.details}, nil
}

// Restore is used to restore an FSM from a snapshot. It is not called
// concurrently with any other command. The FSM must discard all previous
// state.
func (t *Quorum) Restore(r io.ReadCloser) error {
	t.details = Details{}
	return nil
}

// Details includes information about the details of the quorum.
// who its members are, the latest deploys.
func (t Quorum) Details() (d Details, err error) {
	return t.details, nil
}

type quorumSnapshot struct {
	details Details
}

// Persist should dump all necessary state to the WriteCloser 'sink',
// and call sink.Close() when finished or call sink.Cancel() on error.
func (t quorumSnapshot) Persist(sink raft.SnapshotSink) (err error) {
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
func (t quorumSnapshot) Release() {

}
