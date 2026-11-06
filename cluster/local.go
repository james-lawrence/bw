package cluster

import (
	"sync"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/protobuf/proto"
)

type localOption func(*Local)

// LocalOptionCapability sets the capabilities of the local node.
func LocalOptionCapability(c []byte) func(*Local) {
	return func(l *Local) {
		l.Capability = c
	}
}

// NewLocal creates the local node delegate.
func NewLocal(p *agent.Peer, options ...localOption) *Local {
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

	if l.encoded, err = proto.Marshal(l.Peer); err != nil {
		panic(err)
	}

	if l.metadata, err = agent.EncodeMetadata(agent.PeerToMetadata(l.Peer)); err != nil {
		panic(err)
	}

	return &l
}

// Local metadata.
type Local struct {
	Peer       *agent.Peer
	Capability []byte
	encoded    []byte
	m          sync.RWMutex
	metadata   []byte
}

// Metadata returns the encoded metadata gossiped to the cluster.
func (t *Local) Metadata() []byte {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.metadata
}

// SetMode updates the role advertised in the metadata. the new metadata is gossiped the
// next time the node is advertised, see clustering.Replication.
func (t *Local) SetMode(mode agent.PeerMode) (err error) {
	md := agent.PeerToMetadata(t.Peer)
	md.Mode = int32(mode)

	raw, err := agent.EncodeMetadata(md)
	if err != nil {
		return err
	}

	t.m.Lock()
	defer t.m.Unlock()
	t.metadata = raw
	return nil
}
