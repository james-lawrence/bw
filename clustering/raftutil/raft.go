// Package raftutil provides some convience functionality for building
// an internal raft cluster that overlays a cluster of nodes.
package raftutil

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"bitbucket.org/jatone/bearded-wookie/x/contextx"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
)

type cluster interface {
	LocalNode() *memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(int, []byte) []*memberlist.Node
}

type state interface {
	Update(cluster) state
}

type clusterEventType int

const (
	clusterEventJoined = iota
	clusterEventLeft
)

type clusterEvent struct {
	clusterEventType
	Peer *memberlist.Node
}

// ProtocolOption options for the raft protocol.
type ProtocolOption func(*Protocol)

// ProtocolOptionStateMachine set the state machine for the protocol
func ProtocolOptionStateMachine(m func() raft.FSM) ProtocolOption {
	return func(p *Protocol) {
		p.getStateMachine = m
	}
}

// NewProtocol ...
func NewProtocol(ctx context.Context, port uint16, network *raft.NetworkTransport, snaps raft.SnapshotStore, options ...ProtocolOption) (_ignored Protocol, err error) {
	p := Protocol{
		Context:   ctx,
		Port:      port,
		Network:   network,
		Snapshots: snaps,
		getStateMachine: func() raft.FSM {
			return &noopFSM{}
		},
		ClusterChange: sync.NewCond(&sync.Mutex{}),
		init:          &sync.Once{},
		sgroup:        &sync.WaitGroup{},
		events:        make(chan clusterEvent, 100),
	}

	for _, opt := range options {
		opt(&p)
	}

	return p, nil
}

// Protocol - utility data structure for holding information about a raft protocol
// setup that are needed to connect, reconnect, and shutdown.
//
// It cannot be instantiated directly, instead use NewProtocol.
type Protocol struct {
	Context         context.Context
	Port            uint16
	ClusterChange   *sync.Cond
	Snapshots       raft.SnapshotStore
	getStateMachine func() raft.FSM
	Network         *raft.NetworkTransport
	init            *sync.Once
	cluster         *raft.Raft
	sgroup          *sync.WaitGroup
	events          chan clusterEvent
}

// Overlay overlays this raft protocol on top of the provided cluster. blocking.
func (t Protocol) Overlay(c cluster) {
	var (
		s state = passive{
			raftp: &t,
		}
	)
	defer debugx.Println("overlay shutdown")
	t.sgroup.Add(1)
	defer t.sgroup.Done()

	t.sgroup.Add(1)
	go t.background()

	for {
		select {
		case <-t.Context.Done():
			debugx.Println("overlay shutting down")
			return
		default:
			// debugx.Printf("raft state transition %T\n", s)
			s = s.Update(c)
		}
	}
}

func (t Protocol) background() {
	defer t.sgroup.Done()
	for {
		select {
		case <-t.Context.Done():
			return
		case e := <-t.events:
			handleClusterEvent(t.cluster, e)
			t.ClusterChange.Broadcast()
		}
	}
}

func (t Protocol) getPeers(c cluster) []string {
	return peersToString(t.Port, possiblePeers(c)...)
}

// connect - connect to the raft protocol overlay within the given cluster.
func (t *Protocol) connect(c cluster) (*raft.Raft, raft.PeerStore, error) {
	var (
		err      error
		protocol *raft.Raft
	)

	conf := raft.DefaultConfig()
	conf.HeartbeatTimeout = 5 * time.Second
	conf.ElectionTimeout = 10 * time.Second
	conf.MaxAppendEntries = 64
	conf.TrailingLogs = 128
	// ShutdownOnRemove important setting, otherwise peers cannot rejoin the cluster....
	conf.ShutdownOnRemove = false

	store := raft.NewInmemStore()
	speers := &raft.StaticPeers{StaticPeers: t.getPeers(c)}
	if protocol, err = raft.NewRaft(conf, t.getStateMachine(), store, store, t.Snapshots, speers, t.Network); err != nil {
		return nil, nil, err
	}

	protocol.RegisterObserver(raft.NewObserver(nil, false, func(o *raft.Observation) bool {
		log.Printf("raft observation: %T, %#v\n", o.Data, o.Data)
		return false
	}))

	t.init.Do(func() {
		// add this to the parent context waitgroup
		contextx.WaitGroupAdd(t.Context, 1)
		go t.waitShutdown()
		go periodicForceState(t.Context, 1*time.Second, t.ClusterChange)
	})

	t.cluster = protocol

	return protocol, speers, nil
}

func (t Protocol) waitShutdown() {
	defer log.Println("raft protocol clean shutdown")
	defer contextx.WaitGroupDone(t.Context)
	<-t.Context.Done()
	debugx.Println("initiating shutdown for raft protocol")
	// notify the overlay function that something has occurred.
	t.ClusterChange.Broadcast()
	debugx.Println("waiting for overlay to complete")
	// wait for the overlay to complete.
	t.sgroup.Wait()
	debugx.Println("attempting clean shutdown")
	// attempt to cleanly shutdown the local peer.
	t.maybeShutdown()
	debugx.Println("signaling wait group of completion")
}

func (t Protocol) maybeShutdown() {
	if t.cluster == nil {
		return
	}

	if err := t.cluster.Shutdown().Error(); err != nil {
		log.Println("failed to shutdown raft", err)
	}
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (t Protocol) NotifyJoin(peer *memberlist.Node) {
	log.Println("Join:", peer.Name, peer.Addr.String(), int(peer.Port))
	if t.cluster != nil && t.cluster.State() == raft.Leader {
		dup := *peer
		dup.Port = t.Port
		t.events <- clusterEvent{
			Peer:             &dup,
			clusterEventType: clusterEventJoined,
		}
	}
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (t Protocol) NotifyLeave(peer *memberlist.Node) {
	log.Println("Leave:", peer.Name, peer.Addr.String(), int(peer.Port))
	if t.cluster != nil && t.cluster.State() == raft.Leader {
		dup := *peer
		dup.Port = t.Port
		t.events <- clusterEvent{
			Peer:             peer,
			clusterEventType: clusterEventLeft,
		}
	}
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (t Protocol) NotifyUpdate(peer *memberlist.Node) {
	log.Println("Update:", peer.Name, peer.Addr.String(), int(peer.Port))
	t.ClusterChange.Broadcast()
}

func handleClusterEvent(c *raft.Raft, e clusterEvent) {
	if c == nil {
		return
	}

	if c.State() != raft.Leader {
		return
	}

	switch e.clusterEventType {
	case clusterEventJoined:
		if err := c.AddPeer(peerToString(e.Peer.Port, e.Peer)).Error(); err != nil {
			log.Println("failed to add peer", err)
		}
	case clusterEventLeft:
		if err := c.RemovePeer(peerToString(e.Peer.Port, e.Peer)).Error(); err != nil {
			log.Println("failed to remove peer", err)
		}
	}
}

// key used for determining possible candidates for the raft protocol
// within the cluster.
var leaderKey = []byte("leaders")

// Leader - returns the leader of the cluster.
func Leader(c cluster) *memberlist.Node {
	return c.Get(leaderKey)
}

// maybeLeave - uses the provided cluster and raft protocol to determine
// if it should leave the raft protocol group.
// returns true if it left the raft protocol.
func maybeLeave(protocol *raft.Raft, c cluster) bool {
	if isMember(c) {
		return false
	}

	log.Println("no longer a possible member of leadership, leaving raft cluster")
	if err := protocol.Shutdown().Error(); err != nil {
		log.Println("failed to shutdown raft protocol", err)
	}

	return true
}

// isMember utility function for checking if the local node of the cluster is a member
// of the possiblePeers set.
func isMember(c cluster) bool {
	return isPossiblePeer(c.LocalNode(), possiblePeers(c)...)
}

// possiblePeers utility function for locating N possible peers for the raft protocol.
func possiblePeers(c cluster) []*memberlist.Node {
	return c.GetN(3, leaderKey)
}

// isPossiblePeer utility function for determining if the given local node is in
// the set of peers.
func isPossiblePeer(local *memberlist.Node, peers ...*memberlist.Node) bool {
	for _, peer := range peers {
		if local.String() == peer.String() {
			return true
		}
	}

	return false
}

// peersToString ...
func peersToString(port uint16, peers ...*memberlist.Node) []string {
	results := make([]string, 0, len(peers))
	for _, peer := range peers {
		results = append(results, peerToString(port, peer))
	}
	return results
}

// peerToString ...
func peerToString(port uint16, peer *memberlist.Node) string {
	return (&net.TCPAddr{IP: peer.Addr, Port: int(port)}).String()
}

func periodicForceState(ctx context.Context, d time.Duration, c *sync.Cond) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case _ = <-t.C:
			c.Broadcast()
		case _ = <-ctx.Done():
			return
		}
	}
}
