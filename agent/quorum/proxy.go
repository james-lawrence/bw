package quorum

import (
	"context"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewProxyMachine stores the state of the cluster.
func NewProxyMachine(l cluster, r *raft.Raft, d dialers.Defaults) ProxyMachine {
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
	dialer dialers.Defaults
}

// State returns the state of the raft cluster.
func (t *ProxyMachine) State() raft.RaftState {
	return t.state.State()
}

// Leader returns the current leader.
func (t *ProxyMachine) Leader() *agent.Peer {
	return agent.DetectQuorum(t.local, agent.IsLeader(string(t.state.Leader())))
}

func (t *ProxyMachine) leader() (peader *agent.Peer, err error) {
	if p := t.Leader(); p != nil {
		return p, nil
	}

	return nil, status.Error(codes.Unavailable, "failed to locate leader")
}

// DialLeader dials the leader using the given dialer.
func (t *ProxyMachine) DialLeader(d dialers.Defaults) (c *grpc.ClientConn, err error) {
	var (
		leader *agent.Peer
	)

	if leader, err = t.leader(); err != nil {
		return c, err
	}

	return dialers.NewDirect(agent.RPCAddress(leader)).DialContext(context.Background(), d.Defaults()...)
}

func (t *ProxyMachine) Deploy(ctx context.Context, c cluster, dialer dialers.Defaults, by string, dopts *agent.DeployOptions, a *agent.Archive, peers ...*agent.Peer) (err error) {
	var (
		conn *grpc.ClientConn
	)

	if conn, err = t.DialLeader(t.dialer); err != nil {
		return err
	}
	defer conn.Close()

	if _, err = agent.NewQuorumClient(conn).Deploy(ctx, &agent.DeployCommandRequest{Initiator: by, Options: dopts, Archive: a, Peers: peers}); err != nil {
		return err
	}

	return nil
}

// Dispatch a message to the WAL.
func (t *ProxyMachine) Dispatch(ctx context.Context, m ...*agent.Message) (err error) {
	return t.writeWAL(ctx, m...)
}

func (t *ProxyMachine) writeWAL(ctx context.Context, m ...*agent.Message) (err error) {
	var (
		conn *grpc.ClientConn
	)

	if conn, err = t.DialLeader(t.dialer); err != nil {
		return err
	}
	defer conn.Close()

	ctx, done := context.WithTimeout(ctx, 10*time.Second)
	defer done()

	return agent.NewConn(conn).Dispatch(ctx, m...)
}
