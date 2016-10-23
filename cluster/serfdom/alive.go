package serfdom

import (
	"fmt"
	"log"

	"bitbucket.org/jatone/bearded-wookie/cluster"
	"github.com/hashicorp/memberlist"
)

type aliveHandler struct{}

func (aliveHandler) NotifyAlive(peer *memberlist.Node) error {
	if cluster.BitField(peer.Meta).Has(cluster.Lurker) {
		log.Println("NotifyAlive ignoring", peer.Name)
		return fmt.Errorf("ignoring peer: %s", peer.Name)
	}

	return nil
}
