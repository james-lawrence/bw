package raftutil

import (
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
)

type leader struct {
	stateMeta
}

func (t leader) Update(c rendezvous) state {
	var (
		maintainState state = delayed(t, t.protocol.ClusterChange, t.protocol.PassiveCheckin)
	)

	log.Printf("leader update invoked: %p - %s\n", t.r, t.protocol.PassiveCheckin)
	switch t.r.State() {
	case raft.Leader:
		if t.cleanupPeers(t.protocol.LocalNode, agent.QuorumNodes(c)...) {
			refresh := time.Second
			log.Println("peers unstable, will refresh in", refresh)
			return delayed(t, t.protocol.ClusterChange, refresh)
		}

		return maintainState
	default:
		log.Println("lost leadership: demote to peer")
		return peer(t).Update(c)
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
	unmodifiedPeers := config.Configuration().Servers
	voterCount := voterCount(peers)
	allowRemoval := voterCount >= 3

	// we bail out when we fail to add peers because we don't want to remove peers
	// if we failed to add the new peers to the leadership
	for _, peer := range candidates {
		if rs, err = nodeToserver(peer); err != nil {
			log.Println("failed to lookup peer", err)
			return true
		}

		peers = removePeer(rs.ID, peers...)

		if len(strings.TrimSpace(string(rs.Address))) == 0 {
			log.Println("skipping, detected empty address", spew.Sdump(unmodifiedPeers), spew.Sdump(rs))
			continue
		}

		if err = t.r.AddVoter(rs.ID, rs.Address, 0, commitTimeout).Error(); err != nil {
			log.Println("failed to add peer", spew.Sdump(rs), spew.Sdump(unmodifiedPeers), err)
			unstable = true
			continue
		}
	}

	for _, peer := range peers {
		if peer.ID == raft.ServerID(local.Name) {
			continue
		}

		switch peer.Suffrage {
		case raft.Voter:
			if err := t.r.DemoteVoter(peer.ID, 0, commitTimeout).Error(); err != nil {
				log.Println("failed to demote peer", err)
			}
		case raft.Nonvoter:
			// prevent peer removal if minimum number of voters is not met.
			if !allowRemoval {
				log.Println("failed to remove peer, not enough voters")
				continue
			}

			if err := t.r.RemoveServer(peer.ID, 0, commitTimeout).Error(); err != nil {
				log.Println("failed to remove peer", err)
			}
		}

		return true
	}

	if len(peers) > 1 {
		log.Println(local.Name, "preventing leadership transfer too many peers being changed")
		return true
	}

	for _, peer := range peers {
		if peer.ID == raft.ServerID(local.Name) && allowRemoval {
			log.Println(local.Name, "- transferring leadership")
			errorsx.MaybeLog(errors.Wrap(t.r.LeadershipTransfer().Error(), "failed to transfer leadership"))
			return true
		}
	}

	debugx.Println(local.Name, "cluster stable", unstable)
	return unstable
}

func voterCount(peers []raft.Server) (c int) {
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
