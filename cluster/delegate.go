package cluster

import (
	"log"

	"github.com/hashicorp/memberlist"
)

// NewLocal creates the local node delegate.
func NewLocal(metadata []byte) memberlist.Delegate {
	return &local{Metadata: metadata}
}

type local struct {
	Metadata []byte
}

func (t local) NodeMeta(limit int) []byte {
	log.Println("NodeMeta invoked limit:", limit, len(t.Metadata))
	if limit < len(t.Metadata) {
		log.Println("insufficient room to send metadata")
		return []byte(nil)
	}

	return t.Metadata
}

func (t local) LocalState(join bool) []byte {
	return t.Metadata
}

func (t local) GetBroadcasts(overhead, limit int) [][]byte {
	// log.Println("GetBroadcasts invoked overhead:", overhead, "limit:", limit)
	return [][]byte(nil)
}

func (t *local) MergeRemoteState(buf []byte, join bool) {
	log.Println("MergeRemoteState join:", join, "len(buf):", len(buf))
}

func (t local) NotifyMsg(buf []byte) {
	log.Println("NotifyMsg string(buf):", string(buf))
}
