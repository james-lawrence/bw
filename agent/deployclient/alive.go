package deployclient

import (
	"github.com/hashicorp/memberlist"
)

// AliveHandler - alive handler for the cluster.
type AliveHandler struct{}

// NotifyAlive implements the memberlist.AliveDelegate
func (AliveHandler) NotifyAlive(peer *memberlist.Node) (err error) {
	return nil
}
