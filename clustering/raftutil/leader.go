package raftutil

import (
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/debugx"
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

	debugx.Println("leader update invoked")
	switch t.r.State() {
	case raft.Leader:
		// IMPORTANT: an active leader should never leave the raft cluster. changes in leadership
		// are fairly disruptive, but this means the raft cluster can potentially
		// be a single node larger than expected.
		if t.cleanupPeers(c.LocalNode(), agent.QuorumNodes(c)...) {
			refresh := time.Second
			log.Println("peers unstable, will refresh in", refresh)
			return delayedTransition{
				next:     t,
				Duration: refresh,
			}
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

	peers := config.Configuration().Servers
	voterCount := t.voterCount(peers)
	allowRemoval := voterCount >= 3

	// remove self from peer set.
	peers = removePeer(raft.ServerID(local.Name), peers...)

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

	// prevent peer removal if minimum number of voters are not allowed.
	if allowRemoval {
		for _, peer := range peers {
			unstable = true
			switch peer.Suffrage {
			case raft.Voter:
				if err := t.r.DemoteVoter(peer.ID, 0, commitTimeout).Error(); err != nil {
					log.Println("failed to demote peer", err)
				}
			case raft.Nonvoter:
				if err := t.r.RemoveServer(peer.ID, 0, commitTimeout).Error(); err != nil {
					log.Println("failed to demote peer", err)
				}
			}
		}
	}

	return unstable
}

func (t leader) voterCount(peers []raft.Server) (c int) {
	for _, p := range peers {
		if p.Suffrage == raft.Voter {
			c++
		}
	}

	return c
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
