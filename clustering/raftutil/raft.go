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
	"github.com/james-lawrence/bw/internal/x/stringsx"

	"github.com/davecgh/go-spew/spew"
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

type clusterEventType int

const (
	// EventJoined ...
	EventJoined = iota
	// EventLeft ...
	EventLeft
	// EventUpdate ...
	EventUpdate
	// EventUnstable ...
	EventUnstable
)

type clusterObserver interface {
	Observer(Event)
}

// Event ...
type Event struct {
	Type clusterEventType
	Node agent.Peer
	Peer raft.Server
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

// ProtocolOptionSnapshotStorage set the state machine for the protocol
func ProtocolOptionSnapshotStorage(snaps raft.SnapshotStore) ProtocolOption {
	return func(p *Protocol) {
		p.Snapshots = snaps
	}
}

// ProtocolOptionTransport set the state machine for the protocol
func ProtocolOptionTransport(t func() (raft.Transport, error)) ProtocolOption {
	return func(p *Protocol) {
		p.getTransport = t
	}
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

		return raft.NewNetworkTransport(NewTLSStreamLayer(l, ts), 5, 10*time.Second, os.Stderr), nil
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
		p.leadershipGrace = d
	}
}

// NewProtocol ...
func NewProtocol(ctx context.Context, q BacklogQueueWorker, options ...ProtocolOption) (_ignored Protocol, err error) {
	p := Protocol{
		Context:        ctx,
		StabilityQueue: q,
		Snapshots:      raft.NewInmemSnapshotStore(),
		store:          raft.NewInmemStore(),
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
		leadershipGrace:  time.Minute,
		PassiveCheckin:   time.Minute,
	}

	for _, opt := range options {
		opt(&p)
	}

	return p, nil
}

type stateMeta struct {
	initTime  time.Time
	r         *raft.Raft
	transport raft.Transport
	protocol  *Protocol
	sgroup    *sync.WaitGroup
}

// Protocol - utility data structure for holding information about a raft protocol
// setup that are needed to connect, reconnect, and shutdown.
//
// It cannot be instantiated directly, instead use NewProtocol.
type Protocol struct {
	Context          context.Context
	StabilityQueue   BacklogQueueWorker
	ClusterChange    *sync.Cond
	Snapshots        raft.SnapshotStore
	store            *raft.InmemStore
	PassiveCheckin   time.Duration
	getStateMachine  func() raft.FSM
	getTransport     func() (raft.Transport, error)
	transport        raft.Transport
	observers        []*raft.Observer
	clusterObserver  clusterObserver
	enableSingleNode bool
	config           *raft.Config
	leadershipGrace  time.Duration // how long to wait before a missing leader triggers a reset

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
			debugx.Println("overlay shutting down")
			return
		default:
			s = s.Update(c)
		}
	}
}

// RaftAddr ...
func (t *Protocol) RaftAddr(n *memberlist.Node) (raft.Server, error) {
	return t.StabilityQueue.Provider.RaftAddr(n)
}

func (t Protocol) background(sm stateMeta) {
	defer sm.sgroup.Done()
	for {
		select {
		case <-t.Context.Done():
			return
		case e := <-t.StabilityQueue.Queue:
			e.Raft = sm.r
			if t.clusterObserver != nil {
				handleClusterEvent(e, t.clusterObserver)
			} else {
				handleClusterEvent(e)
			}
			t.ClusterChange.Broadcast()
		}
	}
}

func (t Protocol) deadlockedLeadership(local *memberlist.Node, p *raft.Raft, lastSeen time.Time) bool {
	leader := string(p.Leader())

	log.Println(local.Name, "current leader", stringsx.DefaultIfBlank(leader, "[None]"), lastSeen)
	if leader == "" && lastSeen.Add(t.leadershipGrace).Before(time.Now()) {
		log.Println(local.Name, "leader is missing and grace period has passed, resetting this peer", t.leadershipGrace)
		return true
	}

	return false
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
		conf raft.Config
	)

	if network, err = t.getTransport(); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	conf = *t.config
	conf.LocalID = raft.ServerID(c.LocalNode().Name)

	if r, err = raft.NewRaft(&conf, t.getStateMachine(), t.store, t.store, t.Snapshots, network); err != nil {
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

func (t *Protocol) waitShutdown(c cluster, sm stateMeta) {
	defer log.Println(c.LocalNode().Name, "raft protocol clean shutdown")
	defer contextx.WaitGroupDone(t.Context)
	<-t.Context.Done()
	debugx.Println(c.LocalNode().Name, "initiating shutdown for raft protocol")
	// notify the overlay function that something has occurred.
	t.ClusterChange.Broadcast()
	debugx.Println("waiting for overlay to complete")
	// wait for the overlay to complete.
	sm.sgroup.Wait()
	debugx.Println("attempting clean shutdown")
	// attempt to cleanly shutdown the local peer.
	t.maybeShutdown(c, sm.r, sm.transport)
	debugx.Println("signaling wait group of completion")
}

func (t *Protocol) maybeShutdown(c cluster, r *raft.Raft, transport raft.Transport) {
	if r == nil {
		return
	}

	if err := r.Shutdown().Error(); err != nil {
		log.Println(c.LocalNode().Name, "failed to shutdown raft", err)
	}

	if transport == nil {
		return
	}

	if trans, ok := transport.(raft.WithClose); ok {
		log.Println(c.LocalNode().Name, "closed transport", trans.Close())
	}
}

func handleClusterEvent(e Event, obs ...clusterObserver) {
	if e.Raft.State() != raft.Leader {
		return
	}

	switch e.Type {
	case EventJoined:
		if err := agentutil.ApplyToStateMachine(e.Raft, agentutil.NodeEvent(e.Node, agent.Message_Joined), 10*time.Second); err != nil {
			log.Println("failed apply peer", err)
		}
	case EventLeft:
		if err := agentutil.ApplyToStateMachine(e.Raft, agentutil.NodeEvent(e.Node, agent.Message_Departed), 10*time.Second); err != nil {
			log.Println("failed apply peer", err)
		}
	}

	for _, o := range obs {
		o.Observer(e)
	}
}

// QueuedEvent ...
type QueuedEvent struct {
	Event clusterEventType
	Node  *memberlist.Node
}

// BacklogQueueWorker ...
type BacklogQueueWorker struct {
	Provider addressProdvider
	Queue    chan Event
}

// Background ...
func (t BacklogQueueWorker) Background(backlog BacklogQueue) {
	var (
		err error
		rs  raft.Server
	)

	for n := range backlog.Backlog {
		if rs, err = t.Provider.RaftAddr(n.Node); err != nil {
			log.Println("ignoring join due to error decoding meta data", err)
			continue
		}
		p := agent.MustPeer(agent.NodeToPeer(n.Node))

		if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
			log.Println(n.Event, spew.Sdump(p))
		}

		t.Queue <- Event{
			Type: n.Event,
			Node: agent.MustPeer(agent.NodeToPeer(n.Node)),
			Peer: rs,
		}
	}
}

// BacklogQueue ...
type BacklogQueue struct {
	Backlog chan QueuedEvent
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (t BacklogQueue) NotifyJoin(n *memberlist.Node) {
	log.Println("Join:", n.Name)
	select {
	case t.Backlog <- QueuedEvent{Event: EventJoined, Node: n}:
	default:
		log.Println("dropping node join onto the floor due to full queue", n.Name)
	}
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (t BacklogQueue) NotifyLeave(n *memberlist.Node) {
	log.Println("Leave:", n.Name)
	select {
	case t.Backlog <- QueuedEvent{Event: EventLeft, Node: n}:
	default:
		log.Println("dropping node leave onto the floor due to full queue", n.Name)
	}
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (t BacklogQueue) NotifyUpdate(n *memberlist.Node) {
	log.Println("Update:", n.Name)
	select {
	case t.Backlog <- QueuedEvent{Event: EventUpdate, Node: n}:
	default:
		log.Println("dropping node leave onto the floor due to full queue", n.Name)
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

func leave(current state, sm stateMeta) state {
	if err := sm.r.Shutdown().Error(); err != nil {
		log.Println("failed to shutdown raft protocol", err)
		return current
	}

	if sm.transport == nil {
		log.Println("expected a transport to exist during leave but its nil?!")
		return current
	}

	if trans, ok := sm.transport.(raft.WithClose); ok {
		log.Println("closing raft transport")
		if err := trans.Close(); err != nil {
			logx.MaybeLog(errors.WithMessage(err, "failed to close transport"))
			return current
		}
	}

	return passive{
		protocol: sm.protocol,
		sgroup:   sm.sgroup,
	}
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

// // quorumPeers utility function for locating N possible peers for the raft protocol.
// func quorumPeers(c rendezvous) []*memberlist.Node {
// 	return c.GetN(3, []byte(agent.QuorumKey))
// }

func configuration(provider addressProdvider, c cluster) (conf raft.Configuration) {
	var (
		err error
		rs  raft.Server
	)

	for _, peer := range agent.QuorumNodes(c) {
		if rs, err = provider.RaftAddr(peer); err != nil {
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
