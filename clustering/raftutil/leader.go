package raftutil

import (
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
)

type leader struct {
	stateMeta
}

func (t leader) Update(c cluster) state {
	var (
		maintainState state = delayedTransition{
			next:     t,
			Duration: t.protocol.PassiveCheckin,
		}
	)

	log.Println("leader update invoked")
	switch t.r.State() {
	case raft.Leader:
		if t.cleanupPeers(c.LocalNode(), quorumPeers(c)...) {
			go t.protocol.unstable(time.Second)
		}

		if maybeLeave(c) {
			if err := t.r.Barrier(5 * time.Second).Error(); err != nil {
				log.Println("barrier write failed", err)
				return maintainState
			}
			return leave(t, t.stateMeta)
		}
		return maintainState
	default:
		log.Println("lost leadership: leaving")
		return leave(t, t.stateMeta)
	}
}

// cleanupPeers returns true if the peer set was unstable.
func (t leader) cleanupPeers(local *memberlist.Node, candidates ...*memberlist.Node) (unstable bool) {
	const (
		commitTimeout = 10 * time.Second
	)

	var (
		err error
		rs  raft.Server
	)

	if err = t.r.Barrier(commitTimeout).Error(); err != nil {
		log.Println("barrier write failed", err)
		return true
	}

	config := t.r.GetConfiguration()
	if err = config.Error(); err != nil {
		log.Println("failed to retrieve peers", err)
		return true
	}

	// remove self from peer set.
	peers := removePeer(raft.ServerID(local.Name), config.Configuration().Servers...)
	// log.Println(local.Name, "candidates", spew.Sdump(candidates))
	// log.Println(local.Name, "peers", peers)

	// we bail out when we fail to add peers because we don't want to remove peers
	// if we failed to add the new peers to the leadership
	for _, peer := range candidates {
		if rs, err = t.protocol.RaftAddr(peer); err != nil {
			log.Println("failed to lookup peer", err)
			return true
		}

		peers = removePeer(rs.ID, peers...)

		if len(strings.TrimSpace(string(rs.Address))) == 0 {
			log.Println("skipping, detected empty address", spew.Sdump(peer), spew.Sdump(rs))
			continue
		}

		if err = t.r.AddVoter(rs.ID, rs.Address, 0, commitTimeout).Error(); err != nil {
			log.Println("failed to add peer", err)
			return true
		}
	}

	for _, peer := range peers {
		if err := t.r.RemoveServer(peer.ID, t.r.GetConfiguration().Index(), commitTimeout).Error(); err != nil {
			log.Println("failed to remove peer", err)
			unstable = true
		}
	}

	return unstable
}

func removePeer(id raft.ServerID, peers ...raft.Server) []raft.Server {
	result := make([]raft.Server, 0, len(peers))
	for _, peer := range peers {
		if peer.ID == id {
			continue
		}
		result = append(result, peer)
	}

	return result
}
