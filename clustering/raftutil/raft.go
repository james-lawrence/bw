// Package raftutil provides some convience functionality for building
// an internal raft cluster that overlays a cluster of nodes.
package raftutil

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/x/contextx"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/muxer"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type addressProdvider interface {
	RaftAddr(*memberlist.Node) (raft.Server, error)
}

type rendezvous interface {
	GetN(int, []byte) []*memberlist.Node
}

type cluster interface {
	Members() []*memberlist.Node
	LocalNode() *memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(int, []byte) []*memberlist.Node
}

type state interface {
	Update(cluster) state
}

type clusterObserver interface {
	Observer(Event)
}

// Event ...
type Event struct {
	*agent.ClusterWatchEvents
	Raft *raft.Raft
}

// ProtocolOption options for the raft protocol.
type ProtocolOption func(*Protocol)

// ProtocolOptionStateMachine set the state machine for the protocol
func ProtocolOptionStateMachine(m func() raft.FSM) ProtocolOption {
	return func(p *Protocol) {
		p.getStateMachine = m
	}
}

// ProtocolOptionTransport set the state machine for the protocol
func ProtocolOptionTransport(t func() (raft.Transport, error)) ProtocolOption {
	return func(p *Protocol) {
		p.getTransport = t
	}
}

// ProtocolOptionTCPTransport set the state machine for the protocol
func ProtocolOptionMuxerTransport(tcp *net.TCPAddr, m *muxer.M, ts *tls.Config) ProtocolOption {
	return ProtocolOptionTransport(func() (_ raft.Transport, err error) {
		var (
			l net.Listener
			d = muxer.NewDialer(bw.ProtocolRAFT, NewTLSStreamDialer(ts))
		)

		if l, err = m.Bind(bw.ProtocolRAFT, tcp); err != nil {
			return nil, errors.WithStack(err)
		}

		return raft.NewNetworkTransport(NewTLSStreamLayer(l, d), 5, 10*time.Second, os.Stderr), nil
	})
}

// ProtocolOptionTCPTransport set the state machine for the protocol
func ProtocolOptionTCPTransport(tcp *net.TCPAddr, ts *tls.Config) ProtocolOption {
	return ProtocolOptionTransport(func() (raft.Transport, error) {
		var (
			err error
			l   net.Listener
		)

		if ts == nil {
			return raft.NewTCPTransport(tcp.String(), tcp, 5, 10*time.Second, os.Stderr)
		}

		if l, err = net.ListenTCP(tcp.Network(), tcp); err != nil {
			return nil, errors.WithStack(err)
		}

		return raft.NewNetworkTransport(NewTLSStreamLayer(tls.NewListener(l, ts), NewTLSStreamDialer(ts)), 5, 10*time.Second, os.Stderr), nil
	})
}

// ProtocolOptionObservers set the observers for the protocol
func ProtocolOptionObservers(o ...*raft.Observer) ProtocolOption {
	return func(p *Protocol) {
		p.observers = o
	}
}

// ProtocolOptionClusterObserver set the observers for the protocol
func ProtocolOptionClusterObserver(o clusterObserver) ProtocolOption {
	return func(p *Protocol) {
		p.clusterObserver = o
	}
}

// ProtocolOptionEnableSingleNode enables single node operation.
func ProtocolOptionEnableSingleNode(b bool) ProtocolOption {
	return func(p *Protocol) {
		p.enableSingleNode = b
	}
}

// ProtocolOptionConfig set raft configuration for the protocol.
func ProtocolOptionConfig(c *raft.Config) ProtocolOption {
	return func(p *Protocol) {
		p.config = c
	}
}

// ProtocolOptionPassiveCheckin how often to check if the node should have promoted itself.
func ProtocolOptionPassiveCheckin(d time.Duration) ProtocolOption {
	return func(p *Protocol) {
		p.PassiveCheckin = d
	}
}

// ProtocolOptionLeadershipGrace how long to wait before considering the leader dead.
func ProtocolOptionLeadershipGrace(d time.Duration) ProtocolOption {
	return func(p *Protocol) {
		p.lastContactGrace = d
	}
}

// ProtocolOptionPassiveReset reset when we enter the passive state.
func ProtocolOptionPassiveReset(reset func() (Storage, raft.SnapshotStore, error)) ProtocolOption {
	return func(p *Protocol) {
		p.PassiveReset = reset
	}
}

// NewProtocol ...
func NewProtocol(ctx context.Context, q BacklogQueueWorker, options ...ProtocolOption) (_ignored Protocol, err error) {
	p := Protocol{
		Context:        ctx,
		StabilityQueue: q,
		getStateMachine: func() raft.FSM {
			return &noopFSM{}
		},
		getTransport: func() (raft.Transport, error) {
			_, trans := raft.NewInmemTransport("")
			return trans, nil
		},
		ClusterChange:    sync.NewCond(&sync.Mutex{}),
		enableSingleNode: false,
		config:           defaultRaftConfig(),
		lastContactGrace: time.Minute,
		PassiveCheckin:   time.Hour,
		PassiveReset: func() (Storage, raft.SnapshotStore, error) {
			return raft.NewInmemStore(), raft.NewInmemSnapshotStore(), nil
		},
	}

	for _, opt := range options {
		opt(&p)
	}

	return p, nil
}

type stateMeta struct {
	lastContact time.Time
	r           *raft.Raft
	transport   raft.Transport
	protocol    *Protocol
	sgroup      *sync.WaitGroup
	ctx         context.Context
	done        context.CancelFunc
}

func (t *stateMeta) waitShutdown(c cluster) {
	log.Println(c.LocalNode().Name, "raft protocol shutdown initated")
	defer log.Println(c.LocalNode().Name, "raft protocol shutdown completed")
	defer contextx.WaitGroupDone(t.protocol.Context)

	<-t.ctx.Done()

	log.Println(c.LocalNode().Name, "initiating shutdown for raft protocol")
	// attempt to cleanly shutdown the local peer.
	t.cleanShutdown(c)

	log.Println("waiting for background stability queue to complete")

	// wait for the background stability queue to complete.
	t.sgroup.Wait()
}

func (t *stateMeta) cleanShutdown(c cluster) {
	if t.r != nil {
		if err := t.r.Shutdown().Error(); err != nil {
			log.Println(c.LocalNode().Name, "failed to shutdown raft", err)
		}
	}

	if err := autocloseTransport(t.transport); err == nil {
		log.Println(c.LocalNode().Name, "closed transport")
	} else {
		log.Println(c.LocalNode().Name, "failed to close transport", err)
	}
}

func (t *stateMeta) background() {
	defer log.Println("stability queue background shutdown")
	defer t.sgroup.Done()

	for {
		select {
		case <-t.ctx.Done():
			return
		case e := <-t.protocol.StabilityQueue.Queue:
			evt := Event{
				ClusterWatchEvents: e,
				Raft:               t.r,
			}

			if t.protocol.clusterObserver != nil {
				handleClusterEvent(evt, t.protocol.clusterObserver)
			} else {
				handleClusterEvent(evt)
			}

			switch e.Event {
			case agent.ClusterWatchEvents_Update:
			default:
				log.Println("broadcasting cluster change due to unstable cluster", e.Event.String())
				t.protocol.ClusterChange.Broadcast()
			}
		}
	}
}

// Storage ...
type Storage interface {
	raft.LogStore
	raft.StableStore
}

// Protocol - utility data structure for holding information about a raft protocol
// setup that are needed to connect, reconnect, and shutdown.
//
// It cannot be instantiated directly, instead use NewProtocol.
type Protocol struct {
	Context          context.Context
	StabilityQueue   BacklogQueueWorker
	ClusterChange    *sync.Cond
	PassiveReset     func() (Storage, raft.SnapshotStore, error)
	PassiveCheckin   time.Duration
	getStateMachine  func() raft.FSM
	getTransport     func() (raft.Transport, error)
	transport        raft.Transport
	observers        []*raft.Observer
	clusterObserver  clusterObserver
	enableSingleNode bool
	config           *raft.Config
	lastContactGrace time.Duration // how long to wait before a missing leader triggers a reset
}

// Overlay overlays this raft protocol on top of the provided cluster. blocking.
func (t *Protocol) Overlay(c cluster, options ...ProtocolOption) {
	for _, opt := range options {
		opt(t)
	}

	var (
		s state = passive{
			protocol: t,
			sgroup:   &sync.WaitGroup{},
		}
	)

	defer debugx.Println("overlay shutdown")

	for {
		select {
		case <-t.Context.Done():
			log.Println("overlay shutting down")
			return
		default:
			s = s.Update(c)
		}
	}
}

func defaultRaftConfig() *raft.Config {
	conf := raft.DefaultConfig()

	conf.LogOutput = ioutil.Discard
	if envx.Boolean(false, bw.EnvLogsRaft, bw.EnvLogsVerbose) {
		conf.LogOutput = os.Stderr
	}

	conf.LeaderLeaseTimeout = 2 * time.Second
	conf.HeartbeatTimeout = 5 * time.Second
	conf.ElectionTimeout = 10 * time.Second
	conf.SnapshotInterval = 30 * time.Minute
	conf.MaxAppendEntries = 64
	conf.TrailingLogs = 128
	conf.SnapshotThreshold = 256

	return conf
}

// connect - connect to the raft protocol overlay within the given cluster.
func (t *Protocol) connect(c cluster) (network raft.Transport, r *raft.Raft, err error) {
	var (
		conf      raft.Config
		store     Storage
		snapshots raft.SnapshotStore
	)

	if network, err = t.getTransport(); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	conf = *t.config
	conf.LocalID = raft.ServerID(c.LocalNode().Name)

	log.Println("passive resetting raft state")
	if store, snapshots, err = t.PassiveReset(); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	if r, err = raft.NewRaft(&conf, t.getStateMachine(), store, store, snapshots, network); err != nil {
		if failure := logx.MaybeLog(errors.Wrap(autocloseTransport(network), "failed to cleanup")); failure != nil {
			panic(err)
		}
		return nil, nil, errors.WithStack(err)
	}

	r.RegisterObserver(raft.NewObserver(nil, false, func(o *raft.Observation) bool {
		log.Printf("%s - raft observation (broadcasting change): %T, %#v\n", c.LocalNode().Name, o.Data, o.Data)
		t.ClusterChange.Broadcast()
		return false
	}))

	for _, o := range t.observers {
		r.RegisterObserver(o)
	}

	return network, r, nil
}

func autocloseTransport(trans raft.Transport) error {
	if trans == nil {
		return errors.New("missing transport, unable to close")
	}

	if trans, ok := trans.(raft.WithClose); ok {
		return trans.Close()
	}

	log.Printf("unable to close transport, not closable: %T\n", trans)

	return nil
}

func handleClusterEvent(e Event, obs ...clusterObserver) {
	if e.Raft.State() != raft.Leader {
		return
	}

	switch e.Event {
	case agent.ClusterWatchEvents_Joined:
		if err := agentutil.ApplyToStateMachine(e.Raft, agentutil.NodeEvent(e.Node, agent.Message_Joined), 10*time.Second); err != nil {
			log.Println("failed apply peer", err)
		}
	case agent.ClusterWatchEvents_Depart:
		if err := agentutil.ApplyToStateMachine(e.Raft, agentutil.NodeEvent(e.Node, agent.Message_Departed), 10*time.Second); err != nil {
			log.Println("failed apply peer", err)
		}
	}

	for _, o := range obs {
		o.Observer(e)
	}
}

// BacklogQueueWorker ...
type BacklogQueueWorker struct {
	Queue chan *agent.ClusterWatchEvents
}

func (t BacklogQueueWorker) Enqueue(ctx context.Context, evt *agent.ClusterWatchEvents) error {
	if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
		log.Println("BacklogQueueWorker.Enqueue", evt.Event.String(), evt.Node.Ip, evt.Node.Name, "initiated")
		log.Println("BacklogQueueWorker.Enqueue", evt.Event.String(), evt.Node.Ip, evt.Node.Name, "completed")
	}

	select {
	case t.Queue <- evt:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func leave(c cluster, sm stateMeta) state {
	sm.done()

	return passive{
		protocol: sm.protocol,
		sgroup:   &sync.WaitGroup{},
	}
}

// maybeLeave - uses the provided cluster and raft protocol to determine
// if it should leave the raft protocol group.
// returns true if it left the raft protocol.
func maybeLeave(c cluster) bool {
	if isMember(c) {
		return false
	}

	log.Println(c.LocalNode().Name, "no longer a possible member of quorum, leaving raft cluster")
	return true
}

// isMember utility function for checking if the local node of the cluster is a member
// of the possiblePeers set.
func isMember(c cluster) bool {
	return isPossiblePeer(c.LocalNode(), agent.QuorumNodes(c)...)
}

// possiblePeers utility function for locating N possible peers for the raft protocol.
func possiblePeers(n int, c cluster) []*memberlist.Node {
	return c.GetN(n, []byte(agent.QuorumKey))
}

func configuration(c cluster) (conf raft.Configuration) {
	var (
		err error
		rs  raft.Server
	)

	for _, peer := range agent.QuorumNodes(c) {
		if rs, err = nodeToserver(peer); err != nil {
			log.Println("ignoring peer, unable to compute address", peer.String(), err)
			continue
		}

		conf.Servers = append(conf.Servers, rs)
	}

	return conf
}

// isPossiblePeer utility function for determining if the given local node is in
// the set of peers.
func isPossiblePeer(local *memberlist.Node, peers ...*memberlist.Node) bool {
	for _, peer := range peers {
		if local.Address() == peer.Address() {
			return true
		}
	}

	return false
}

// peersToString ...
func peersToString(peers ...*memberlist.Node) []string {
	results := make([]string, 0, len(peers))
	for _, peer := range peers {
		results = append(results, peer.Name)
	}
	return results
}

func nodeToserver(n *memberlist.Node) (_zero raft.Server, err error) {
	var (
		p *agent.Peer
	)

	if p, err = agent.NodeToPeer(n); err != nil {
		return _zero, err
	}

	return raft.Server{
		ID:       raft.ServerID(n.Name),
		Address:  raft.ServerAddress(agent.RaftAddress(p)),
		Suffrage: raft.Voter,
	}, err
}
