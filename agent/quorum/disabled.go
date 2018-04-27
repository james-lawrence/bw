package quorum

import (
	"context"
	"errors"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
)

// ErrDisabledMachine returned when the state machine interface is disabled.
var ErrDisabledMachine = errors.New("this node is not a member of the quorum")

// DisabledMachine implements the machine api but errors out or
// returns reasonable results on every method.
type DisabledMachine struct {
	agent.EventBus
}

// State returns the state of the raft cluster.
func (t DisabledMachine) State() raft.RaftState {
	return raft.Shutdown
}

// Leader returns the current leader.
func (t DisabledMachine) Leader() (peader agent.Peer, err error) {
	return peader, ErrDisabledMachine
}

// DialLeader dials the leader using the given dialer.
func (t DisabledMachine) DialLeader(d agent.Dialer) (c agent.Client, err error) {
	return c, ErrDisabledMachine
}

// Dispatch a message to the WAL.
func (t DisabledMachine) Dispatch(_ context.Context, m ...agent.Message) (err error) {
	return ErrDisabledMachine
}

// Deploy write a deploy command into the logs
func (t DisabledMachine) Deploy(dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) (err error) {
	return ErrDisabledMachine
}

// Cancel cancel a ongoing deploy.
func (t DisabledMachine) Cancel() error {
	return ErrDisabledMachine
}
