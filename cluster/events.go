package cluster

import (
	"log"

	"github.com/hashicorp/memberlist"
)

// LoggingEventHandler ...
type LoggingEventHandler struct{}

// NotifyJoin logs when a peer joins.
func (t LoggingEventHandler) NotifyJoin(peer *memberlist.Node) {
	log.Println("NotifyJoin", peer.Name)
}

// NotifyLeave logs when a peer leaves.
func (t LoggingEventHandler) NotifyLeave(peer *memberlist.Node) {
	log.Println("NotifyLeave", peer.Name)
}

// NotifyUpdate logs when a peer updates.
func (t LoggingEventHandler) NotifyUpdate(peer *memberlist.Node) {
	log.Println("NotifyUpdate", peer.Name)
}
