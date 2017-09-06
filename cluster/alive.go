package cluster

import (
	"fmt"
	"log"

	"github.com/hashicorp/memberlist"
)

// AliveDefault - default alive handler for the cluster.
// ignores nodes with the Lurker bit set.
type AliveDefault struct{}

// NotifyAlive implements the memberlist.AliveDelegate
func (AliveDefault) NotifyAlive(peer *memberlist.Node) error {
	if BitField(peer.Meta).Has(Deploy) {
		log.Println("NotifyAlive ignoring", peer.Name)
		return fmt.Errorf("ignoring peer: %s", peer.Name)
	}

	return nil
}
