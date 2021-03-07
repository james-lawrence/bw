package cluster

import (
	"github.com/golang/protobuf/proto"
	"github.com/james-lawrence/bw/agent"
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
	metadata   []byte
}
