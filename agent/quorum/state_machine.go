package quorum

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
)

// Initializer for the state machine.
type Initializer interface {
	Initialize(agent.Dispatcher) error
}

// NewMachine ...
func NewMachine(l *agent.Peer, rp *raft.Raft, inits ...Initializer) *StateMachine {
	return &StateMachine{
		l:     l,
		state: rp,
		inits: inits,
	}
}

// StateMachine wraps the raft protocol giving more convient access to the protocol.
type StateMachine struct {
	l     *agent.Peer
	state *raft.Raft
	inits []Initializer
}

func (t *StateMachine) initialize() (err error) {
	for _, init := range t.inits {
		if err = init.Initialize(t); err != nil {
			return err
		}
	}

	return nil
}

// Leader returns the current leader.
func (t *StateMachine) Leader() *agent.Peer {
	return t.l
}

// State returns the state of the raft cluster.
func (t *StateMachine) State() raft.RaftState {
	return t.state.State()
}

// Dispatch a message to the WAL.
func (t *StateMachine) Dispatch(ctx context.Context, messages ...*agent.Message) (err error) {
	for _, m := range messages {
		if err = t.writeWAL(m, 10*time.Second); err != nil {
			return err
		}
	}

	return nil
}

func (t *StateMachine) writeWAL(m *agent.Message, d time.Duration) (err error) {
	var (
		encoded []byte
		future  raft.ApplyFuture
		ok      bool
	)

	if encoded, err = proto.Marshal(m); err != nil {
		return errors.WithStack(err)
	}

	// write the event to the WAL.
	future = t.state.Apply(encoded, 10*time.Second)

	if err = future.Error(); err != nil {
		return errors.WithStack(err)
	}

	if err, ok = future.Response().(error); ok {
		return errors.WithStack(err)
	}

	return err
}
