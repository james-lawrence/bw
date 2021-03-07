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
type DisabledMachine struct{}

// Leader returns the leader peer of the cluster.
func (t DisabledMachine) Leader() *agent.Peer { return nil }

// State returns the state of the raft cluster.
func (t DisabledMachine) State() raft.RaftState {
	return raft.Shutdown
}

// Dispatch a message to the WAL.
func (t DisabledMachine) Dispatch(_ context.Context, m ...*agent.Message) (err error) {
	return ErrDisabledMachine
}
