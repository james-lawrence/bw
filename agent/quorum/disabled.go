package quorum

import (
	"context"

	"github.com/hashicorp/raft"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
)

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
	return status.Error(codes.Unavailable, agent.ErrDisabledMachine.Error())
}

func (t DisabledMachine) Deploy(ctx context.Context, c cluster, dialer dialers.Defaults, by string, dopts *agent.DeployOptions, a *agent.Archive, peers ...*agent.Peer) (err error) {
	return status.Error(codes.Unavailable, agent.ErrDisabledMachine.Error())
}
