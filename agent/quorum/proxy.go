package quorum

import (
	"errors"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
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
func (t *ProxyMachine) Dispatch(m ...agent.Message) (err error) {
	return t.writeWAL(m...)
}

// Deploy trigger a deploy
func (t *ProxyMachine) Deploy(dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) (err error) {
	var (
		c agent.Client
	)

	if c, err = t.DialLeader(t.dialer); err != nil {
		return err
	}

	defer c.Close()
	return c.RemoteDeploy(dopts, a, peers...)
}

// Cancel cancel a ongoing deploy.
func (t *ProxyMachine) Cancel() error {
	dc := agent.DeployCommand{Command: agent.DeployCommand_Cancel}
	return t.writeWAL(agentutil.DeployCommand(t.local.Local(), dc))
}

func (t *ProxyMachine) writeWAL(m ...agent.Message) (err error) {
	var (
		c agent.Client
	)

	if c, err = t.DialLeader(t.dialer); err != nil {
		return err
	}

	defer c.Close()
	return c.Dispatch(m...)
}
