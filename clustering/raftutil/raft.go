// Package raftutil provides some convience functionality for building
// an internal raft cluster that overlays a cluster of nodes.
package raftutil

import (
	"context"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"bitbucket.org/jatone/bearded-wookie/x/contextx"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
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
	// EventJoined ...
	EventJoined = iota
	// EventLeft ...
	EventLeft
	// EventUnstable ...
	EventUnstable
)

type clusterObserver interface {
	Observer(Event)
}

// Event ...
type Event struct {
	Type clusterEventType
	Peer *memberlist.Node
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

// NewProtocol ...
func NewProtocol(ctx context.Context, port uint16, options ...ProtocolOption) (_ignored Protocol, err error) {
	p := Protocol{
		Context:   ctx,
		Port:      port,
		Snapshots: raft.NewDiscardSnapshotStore(),
		getStateMachine: func() raft.FSM {
			return &noopFSM{}
		},
		getTransport: func() (raft.Transport, error) {
			_, trans := raft.NewInmemTransport("")
			return trans, nil
		},
		ClusterChange:    sync.NewCond(&sync.Mutex{}),
		init:             &sync.Once{},
		sgroup:           &sync.WaitGroup{},
		events:           make(chan Event, 100),
		enableSingleNode: false,
		config:           defaultRaftConfig(),
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
	Context          context.Context
	Port             uint16
	ClusterChange    *sync.Cond
	Snapshots        raft.SnapshotStore
	getStateMachine  func() raft.FSM
	getTransport     func() (raft.Transport, error)
	init             *sync.Once
	cluster          *raft.Raft
	observers        []*raft.Observer
	sgroup           *sync.WaitGroup
	clusterObserver  clusterObserver
	events           chan Event
	enableSingleNode bool
	config           *raft.Config
}

// Raft returns the underlying raft instance, can be nil.
func (t *Protocol) Raft() *raft.Raft {
	return t.cluster
}

// Overlay overlays this raft protocol on top of the provided cluster. blocking.
func (t *Protocol) Overlay(c cluster, options ...ProtocolOption) {
	for _, opt := range options {
		opt(t)
	}

	var (
		s state = passive{
			raftp: t,
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
			if t.clusterObserver != nil {
				handleClusterEvent(e, t.clusterObserver)
			}
			t.ClusterChange.Broadcast()
		}
	}
}

func (t Protocol) getPeers(c cluster) []string {
	return peersToString(t.Port, possiblePeers(c)...)
}

func (t Protocol) unstable(d time.Duration) {
	log.Println("peers unstable, will refresh in", d)
	time.Sleep(d)
	t.ClusterChange.Broadcast()
}

func defaultRaftConfig() *raft.Config {
	conf := raft.DefaultConfig()

	conf.HeartbeatTimeout = 5 * time.Second
	conf.ElectionTimeout = 10 * time.Second
	conf.MaxAppendEntries = 64
	conf.TrailingLogs = 128

	return conf
}

// connect - connect to the raft protocol overlay within the given cluster.
func (t *Protocol) connect(c cluster) (*raft.Raft, error) {
	var (
		err      error
		protocol *raft.Raft
		network  raft.Transport
		conf     raft.Config
	)

	if network, err = t.getTransport(); err != nil {
		return nil, errors.WithStack(err)
	}

	conf = *t.config
	conf.LocalID = raft.ServerID(c.LocalNode().Name)

	store := raft.NewInmemStore()
	if protocol, err = raft.NewRaft(&conf, t.getStateMachine(), store, store, t.Snapshots, network); err != nil {
		return nil, err
	}

	protocol.RegisterObserver(raft.NewObserver(nil, false, func(o *raft.Observation) bool {
		log.Printf("raft observation (broadcasting change): %T, %#v\n", o.Data, o.Data)
		t.ClusterChange.Broadcast()
		return false
	}))

	for _, o := range t.observers {
		protocol.RegisterObserver(o)
	}

	t.init.Do(func() {
		// add this to the parent context waitgroup
		contextx.WaitGroupAdd(t.Context, 1)
		go t.waitShutdown(network)
	})

	t.cluster = protocol

	return protocol, nil
}

func (t *Protocol) waitShutdown(transport raft.Transport) {
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

	if trans, ok := transport.(raft.WithClose); ok {
		log.Println("closed transport", trans.Close())
	}
	debugx.Println("signaling wait group of completion")
}

func (t *Protocol) maybeShutdown() {
	if t.cluster == nil {
		return
	}

	if err := t.cluster.Shutdown().Error(); err != nil {
		log.Println("failed to shutdown raft", err)
	}
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (t *Protocol) NotifyJoin(peer *memberlist.Node) {
	log.Println("Join:", peer.Name, net.JoinHostPort(peer.Addr.String(), strconv.Itoa(int(t.Port))))
	t.ClusterChange.Broadcast()
	if t.cluster != nil && t.cluster.State() == raft.Leader {
		dup := *peer
		dup.Port = t.Port
		t.events <- Event{
			Peer: &dup,
			Type: EventJoined,
			Raft: t.cluster,
		}
	}
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (t *Protocol) NotifyLeave(peer *memberlist.Node) {
	log.Println("Leave:", peer.Name, net.JoinHostPort(peer.Addr.String(), strconv.Itoa(int(t.Port))))
	t.ClusterChange.Broadcast()
	if t.cluster != nil && t.cluster.State() == raft.Leader {
		dup := *peer
		dup.Port = t.Port
		t.events <- Event{
			Peer: peer,
			Type: EventLeft,
			Raft: t.cluster,
		}
	}
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (t *Protocol) NotifyUpdate(peer *memberlist.Node) {
	log.Println("Update:", peer.Name, net.JoinHostPort(peer.Addr.String(), strconv.Itoa(int(t.Port))))
	t.ClusterChange.Broadcast()
}

func handleClusterEvent(e Event, obs ...clusterObserver) {
	if e.Raft.State() != raft.Leader {
		return
	}

	config := e.Raft.GetConfiguration()
	if err := config.Error(); err != nil {
		log.Println("failed to retrieve configuration", err)
		return
	}

	switch e.Type {
	case EventJoined:
		p := peerToString(e.Peer.Port, e.Peer)
		future := e.Raft.AddVoter(raft.ServerID(e.Peer.Name), raft.ServerAddress(p), config.Index(), time.Second)
		if err := future.Error(); err != nil {
			log.Println("failed to add peer", err)
		}
	case EventLeft:
		future := e.Raft.RemoveServer(raft.ServerID(e.Peer.Name), config.Index(), time.Second)
		if err := future.Error(); err != nil {
			log.Println("failed to remove peer", err)
		}
	}

	for _, o := range obs {
		o.Observer(e)
	}
}

// key used for determining possible candidates for the raft protocol
// within the cluster.
var leaderKey = []byte("leaders")

// Quorum - returns the leader of the cluster.
func Quorum(c cluster) []*memberlist.Node {
	return possiblePeers(c)
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

func maybeBootstrap(port uint16, protocol *raft.Raft, c cluster) {
	if protocol.Leader() != "" {
		return
	}

	log.Println("attempting a bootstrap refreshing peers")

	if err := protocol.BootstrapCluster(configuration(port, c)).Error(); err != nil {
		log.Println("bootstrap failed", err)
		return
	}
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

func configuration(port uint16, c cluster) (conf raft.Configuration) {
	for _, peer := range possiblePeers(c) {
		p := raft.ServerAddress(peerToString(port, peer))
		conf.Servers = append(conf.Servers, raft.Server{ID: raft.ServerID(peer.Name), Suffrage: raft.Voter, Address: p})
	}

	return conf
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
