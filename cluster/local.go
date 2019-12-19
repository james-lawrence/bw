package cluster

import (
	"log"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/envx"

	"github.com/golang/protobuf/proto"
)

type localOption func(*Local)

// LocalOptionCapability sets the capabilities of the local node.
func LocalOptionCapability(c []byte) func(*Local) {
	return func(l *Local) {
		l.Capability = c
	}
}

// NewLocal creates the local node delegate.
func NewLocal(p agent.Peer, options ...localOption) Local {
	var (
		err error
	)

	l := Local{
		Peer:       p,
		Capability: []byte{},
	}

	for _, opt := range options {
		opt(&l)
	}

	m := agent.PeerMetadata{
		Status:        int32(l.Peer.Status),
		Capability:    l.Capability,
		RPCPort:       l.Peer.RPCPort,
		RaftPort:      l.Peer.RaftPort,
		SWIMPort:      l.Peer.SWIMPort,
		TorrentPort:   l.Peer.TorrentPort,
		DiscoveryPort: l.Peer.DiscoveryPort,
		AutocertPort:  l.Peer.AutocertPort,
	}

	if l.metadata, err = proto.Marshal(&m); err != nil {
		panic(err)
	}

	return l
}

// Local metadata.
type Local struct {
	Peer       agent.Peer
	Capability []byte
	metadata   []byte
}

// NodeMeta provides the metadata about the node.
func (t Local) NodeMeta(limit int) []byte {
	log.Println("NodeMeta invoked limit:", limit, len(t.metadata))
	if limit < len(t.metadata) {
		log.Println("insufficient room to send metadata")
		return []byte(nil)
	}

	return t.metadata
}

// LocalState ...
func (t Local) LocalState(join bool) []byte {
	return t.Capability
}

// GetBroadcasts ...
func (t Local) GetBroadcasts(overhead, limit int) [][]byte {
	return [][]byte(nil)
}

// MergeRemoteState ...
func (t Local) MergeRemoteState(buf []byte, join bool) {
	if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
		log.Println("MergeRemoteState join:", join, "len(buf):", len(buf))
	}
}

// NotifyMsg ...
func (t Local) NotifyMsg(buf []byte) {
	if envx.Boolean(false, bw.EnvLogsGossip, bw.EnvLogsVerbose) {
		log.Println("NotifyMsg string(buf):", string(buf))
	}
}
