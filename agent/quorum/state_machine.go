package quorum

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/pkg/errors"
)

const (
	none int32 = iota
	deploying
)

// StateMachineOption options for the state machine.
type StateMachineOption func(*StateMachine)

// NewStateMachine stores the state of the cluster.
func NewStateMachine(l cluster, r *raft.Raft, d agent.Dialer, deploy deployer, options ...StateMachineOption) StateMachine {
	sm := StateMachine{
		local:    l,
		state:    r,
		dialer:   d,
		deployer: deploy,
	}

	for _, opt := range options {
		opt(&sm)
	}

	return sm
}

// StateMachine encapsulates the details of the raft cluster. granting a type safe API
// to the underlying state machine.
type StateMachine struct {
	deployer
	local  cluster
	state  *raft.Raft
	dialer agent.Dialer
}

// State returns the state of the raft cluster.
func (t *StateMachine) State() raft.RaftState {
	return t.state.State()
}

// Leader returns the current leader.
func (t *StateMachine) Leader() (peader agent.Peer, err error) {
	for _, peader = range t.local.Peers() {
		if agent.RaftAddress(peader) == string(t.state.Leader()) {
			return peader, err
		}
	}

	return peader, errors.New("failed to locate leader")
}

// DialLeader dials the leader using the given dialer.
func (t *StateMachine) DialLeader(d agent.Dialer) (c agent.Client, err error) {
	var (
		leader agent.Peer
	)

	if leader, err = t.Leader(); err != nil {
		return c, err
	}

	return d.Dial(leader)
}

// Dispatch a message to the WAL.
func (t *StateMachine) Dispatch(ctx context.Context, messages ...agent.Message) (err error) {
	for _, m := range messages {
		if err = t.writeWAL(m, 10*time.Second); err != nil {
			return err
		}
	}

	return nil
}

// Deploy trigger a deploy.
func (t *StateMachine) Deploy(dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) (err error) {
	debugx.Println("deploy command initiated")
	defer debugx.Println("deploy command completed")
	return t.deployer.Deploy(t.dialer, t, dopts, a, peers...)
}

// Cancel cancel a ongoing deploy.
func (t *StateMachine) Cancel() error {
	dc := agent.DeployCommand{Command: agent.DeployCommand_Cancel}
	return t.writeWAL(agentutil.DeployCommand(t.local.Local(), dc), 10*time.Second)
}

func (t *StateMachine) writeWAL(m agent.Message, d time.Duration) (err error) {
	var (
		encoded []byte
		future  raft.ApplyFuture
	)

	if encoded, err = proto.Marshal(&m); err != nil {
		return errors.WithStack(err)
	}

	// write the event to the WAL.
	if future = t.state.Apply(encoded, 10*time.Second); future.Error() != nil {
		return errors.WithStack(future.Error())
	}

	return nil
}
