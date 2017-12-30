package cluster

import (
	"fmt"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

// AliveDefault - default alive handler for the cluster.
// ignores nodes with the Lurker bit set.
type AliveDefault struct{}

// NotifyAlive implements the memberlist.AliveDelegate
func (AliveDefault) NotifyAlive(peer *memberlist.Node) (err error) {
	var (
		m Metadata
	)

	if err = proto.Unmarshal(peer.Meta, &m); err != nil {
		log.Println("failed to decode metadata", err)
		return errors.WithStack(err)
	}

	if BitField(m.Capability).Has(Passive) {
		return fmt.Errorf("ignoring peer: %s", peer.Name)
	}

	return nil
}
