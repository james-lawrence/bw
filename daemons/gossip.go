package daemons

import (
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"
)

type notaryStorage interface {
	Lookup(fingerprint string) (g notary.Grant, err error)
	Insert(notary.Grant) (notary.Grant, error)
	Delete(notary.Grant) (notary.Grant, error)
}

type gossipOption func(*Gossip)

// GossipOptionCapability sets the capabilities of the local node.
func GossipOptionCapability(c []byte) func(*Gossip) {
	return func(l *Gossip) {
		l.Capability = c
	}
}

// GossipOptionAuthorization the public key for this node.
func GossipOptionAuthorization(b []byte) func(*Gossip) {
	return func(l *Gossip) {
		l.Authorization = b
	}
}

// GossipOptionNotary storage for the authorization.
func GossipOptionNotary(ns notaryStorage) func(*Gossip) {
	return func(l *Gossip) {
		l.NotaryStorage = ns
	}
}

// NewGossip creates the local node delegate.
func NewGossip(p agent.Peer, options ...gossipOption) Gossip {
	var (
		err error
	)

	l := Gossip{
		Peer:       p,
		Capability: []byte{},
	}

	for _, opt := range options {
		opt(&l)
	}

	m1 := agent.PeerMetadata{
		Status:        int32(l.Peer.Status),
		Capability:    l.Capability,
		RPCPort:       l.Peer.RPCPort,
		RaftPort:      l.Peer.RaftPort,
		SWIMPort:      l.Peer.SWIMPort,
		TorrentPort:   l.Peer.TorrentPort,
		DiscoveryPort: l.Peer.DiscoveryPort,
		AutocertPort:  l.Peer.AutocertPort,
	}

	if l.metadata, err = proto.Marshal(&m1); err != nil {
		panic(err)
	}

	m2 := agent.PeerState{
		Metadata:      &m1,
		Authorization: l.Authorization,
	}

	if l.encodedState, err = proto.Marshal(&m2); err != nil {
		panic(err)
	}

	return l
}

// Gossip used to advertise local metadata and merge remote state.
type Gossip struct {
	Peer          agent.Peer
	NotaryStorage notaryStorage
	Capability    []byte
	Authorization []byte
	metadata      []byte
	encodedState  []byte
}

// NodeMeta provides the metadata about the node.
func (t Gossip) NodeMeta(limit int) []byte {
	if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
		log.Println("NodeMeta:", len(t.metadata), ">", limit)
	}

	if len(t.metadata) > limit {
		return []byte(nil)
	}

	return t.metadata
}

// LocalState includes data like the authorization key.
func (t Gossip) LocalState(join bool) []byte {
	return t.encodedState
}

// GetBroadcasts ...
func (t Gossip) GetBroadcasts(overhead, limit int) [][]byte {
	return [][]byte(nil)
}

// MergeRemoteState ...
func (t Gossip) MergeRemoteState(buf []byte, join bool) {
	var (
		m agent.PeerState
	)

	if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
		log.Println("MergeRemoteState join:", join, "len(buf):", len(buf))
	}

	if err := proto.Unmarshal(buf, &m); err != nil {
		log.Printf("%+v\n", errors.Wrap(err, "merging remote state failed"))
		return
	}

	if _, err := t.NotaryStorage.Insert(notary.Grant{Authorization: m.Authorization}); err != nil {
		log.Println(errors.Wrap(err, "persisting state failed"))
		return
	}
}

// NotifyMsg ...
func (t Gossip) NotifyMsg(buf []byte) {
	if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
		log.Println("NotifyMsg len(buf):", len(buf))
	}
}
