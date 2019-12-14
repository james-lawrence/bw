package quorum

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
)

// NewProxyMachine stores the state of the cluster.
func NewProxyMachine(l cluster, r *raft.Raft, d agent.Dialer) ProxyMachine {
	sm := ProxyMachine{
		local:  l,
		state:  r,
		dialer: d,
	}

	return sm
}

// ProxyMachine a proxy to the state machine, used on follower nodes to proxy commands
// to the leader.
type ProxyMachine struct {
	local  cluster
	state  *raft.Raft
	dialer agent.Dialer
}

// State returns the state of the raft cluster.
func (t *ProxyMachine) State() raft.RaftState {
	return t.state.State()
}

// Leader returns the current leader.
func (t *ProxyMachine) Leader() (peader agent.Peer, err error) {
	for _, peader = range t.local.Peers() {
		if agent.RaftAddress(peader) == string(t.state.Leader()) {
			return peader, err
		}
	}

	return peader, errors.New("failed to locate leader")
}

// DialLeader dials the leader using the given dialer.
func (t *ProxyMachine) DialLeader(d agent.Dialer) (c agent.Client, err error) {
	var (
		leader agent.Peer
	)

	if leader, err = t.Leader(); err != nil {
		return c, err
	}

	return d.Dial(leader)
}

// Dispatch a message to the WAL.
func (t *ProxyMachine) Dispatch(_ context.Context, m ...agent.Message) (err error) {
	return t.writeWAL(m...)
}

func (t *ProxyMachine) writeWAL(m ...agent.Message) (err error) {
	var (
		c agent.Client
	)

	if c, err = t.DialLeader(t.dialer); err != nil {
		return err
	}

	defer c.Close()
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()

	return c.Dispatch(ctx, m...)
}
