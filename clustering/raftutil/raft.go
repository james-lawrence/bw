// Package raftutil provides some convience functionality for building
// an internal raft cluster that overlays a cluster of nodes.
package raftutil

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"

	"bitbucket.org/jatone/bearded-wookie/x/debugx"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type cluster interface {
	LocalNode() *memberlist.Node
	GetN(int, []byte) []*memberlist.Node
}

type state interface {
	Update(cluster) state
}

// NewProtocol ...
func NewProtocol(ctx context.Context, inet *net.TCPAddr, snaps raft.SnapshotStore) (_ignored Protocol, err error) {
	var (
		network *raft.NetworkTransport
	)

	if network, err = raft.NewTCPTransport(inet.String(), inet, 3, time.Second, ioutil.Discard); err != nil {
		return _ignored, errors.Wrap(err, "failed to build raft transport")
	}

	return Protocol{
		Context:       ctx,
		Address:       inet,
		Network:       network,
		Snapshots:     snaps,
		ClusterChange: sync.NewCond(&sync.Mutex{}),
		NotifyCh:      make(chan bool),
		init:          &sync.Once{},
		sgroup:        &sync.WaitGroup{},
	}, nil
}

// Protocol - utility data structure for holding information about a raft protocol
// setup that are needed to connect, reconnect, and shutdown.
//
// It cannot be instantiated directly, instead use NewProtocol.
type Protocol struct {
	Context       context.Context
	NotifyCh      chan bool
	ClusterChange *sync.Cond
	Address       *net.TCPAddr
	Snapshots     raft.SnapshotStore
	Network       *raft.NetworkTransport
	init          *sync.Once
	cluster       *raft.Raft
	sgroup        *sync.WaitGroup
}

// Overlay overlays this raft protocol on top of the provided cluster. blocking.
func (t Protocol) Overlay(c cluster) {
	var (
		s state = passive{
			raftp: t,
		}
	)
	defer debug.Println("overlay shutdown")
	t.sgroup.Add(1)
	defer t.sgroup.Done()

	for {
		select {
		case <-t.Context.Done():
			debug.Println("overlay shutting down")
			return
		default:
			s = s.Update(c)
		}
	}
}

// connect - connect to the raft protocol overlay within the given cluster.
func (t Protocol) connect(c cluster) (*raft.Raft, raft.PeerStore, error) {
	var (
		err      error
		protocol *raft.Raft
	)

	conf := raft.DefaultConfig()
	conf.HeartbeatTimeout = 5 * time.Second
	conf.ElectionTimeout = 10 * time.Second
	conf.MaxAppendEntries = 64
	conf.TrailingLogs = 128
	conf.NotifyCh = t.NotifyCh
	// ShutdownOnRemove important setting, otherwise peers cannot rejoin the cluster....
	conf.ShutdownOnRemove = false

	store := raft.NewInmemStore()
	fsm := &noopFSM{}
	speers := &raft.StaticPeers{StaticPeers: peersToString(t.Address.Port, possiblePeers(c)...)}
	if protocol, err = raft.NewRaft(conf, fsm, store, store, t.Snapshots, speers, t.Network); err != nil {
		return nil, nil, err
	}

	t.init.Do(func() {
		go t.waitShutdown()
	})

	return protocol, speers, nil
}

func (t Protocol) waitShutdown() {
	defer log.Println("raft protocol clean shutdown")
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
	debugx.Println("closing notification channel")
	// finally shutdown leadership channel.
	close(t.NotifyCh)
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

// key used for determining possible candidates for the raft protocol
// within the cluster.
var leaderKey = []byte("leaders")

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
	local := c.LocalNode()
	peers := possiblePeers(c)

	return isPossiblePeer(local, peers...)
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
func peersToString(port int, peers ...*memberlist.Node) []string {
	results := make([]string, 0, len(peers))
	for _, peer := range peers {
		results = append(results, peerToString(port, peer))
	}
	return results
}

// peerToString ...
func peerToString(port int, peer *memberlist.Node) string {
	return (&net.TCPAddr{IP: peer.Addr, Port: port}).String()
}
