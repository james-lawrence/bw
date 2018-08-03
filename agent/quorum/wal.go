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
	"github.com/james-lawrence/bw/x/debugx"
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

// WAL for the quorum.
type WAL struct {
	logs      []agent.Message
	observer  chan agent.Message
	m         *sync.RWMutex
	deploying int32 // is a deploy process in progress.
	// lastSuccessfulDeploy // used for bootstrapping and recovering when a deploy proxy fails.
	// currentDeploy // currently active deploy.
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

	// TODO consider moving observer into the state machine, would resolve this issue.
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
	default:
		atomic.SwapInt32(&t.deploying, none)
	}

	return nil
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

	t.logs = make([]agent.Message, 0, len(decoded.Messages))
	t.deploying = none
	for _, m := range decoded.Messages {
		tmp := *m
		t.logs = append(t.logs, tmp)
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
