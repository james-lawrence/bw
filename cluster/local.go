package cluster

import (
	"log"

	"github.com/golang/protobuf/proto"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
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

	m := Metadata{
		Capability: l.Capability,
		RPCPort:    l.Peer.RPCPort,
		RaftPort:   l.Peer.RaftPort,
		SWIMPort:   l.Peer.SWIMPort,
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
	// log.Println("GetBroadcasts invoked overhead:", overhead, "limit:", limit)
	return [][]byte(nil)
}

// MergeRemoteState ...
func (t Local) MergeRemoteState(buf []byte, join bool) {
	log.Println("MergeRemoteState join:", join, "len(buf):", len(buf))
}

// NotifyMsg ...
func (t Local) NotifyMsg(buf []byte) {
	log.Println("NotifyMsg string(buf):", string(buf))
}