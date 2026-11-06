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
)

// defaults for the stateMeta leadership settings.
const (
	defaultBarrierTimeout     = 10 * time.Second
	defaultMaxBarrierFailures = 3
	defaultStepdownTimeout    = 5 * time.Second
)

type leader struct {
	stateMeta
	barrierFailures int // consecutive barrier failures.
}

func (t leader) Update(c rendezvous) state {
	log.Printf("leader update invoked: %p - %s\n", t.r, t.protocol.PassiveCheckin)
	switch t.r.State() {
	case raft.Leader:
		if err := t.r.Barrier(t.barrierTimeout).Error(); err != nil {
			log.Println("barrier write failed", err)

			t.barrierFailures++
			if t.barrierFailures >= t.maxBarrierFailures {
				return t.stepdown(c, err)
			}

			refresh := time.Second
			log.Println("peers unstable, will refresh in", refresh)
			return delayed(t, t.protocol.ClusterChange, refresh)
		}

		t.barrierFailures = 0
		maintainState := delayed(t, t.protocol.ClusterChange, t.protocol.PassiveCheckin)

		if t.cleanupPeers(t.protocol.LocalNode, agent.QuorumNodes(c)...) {
			refresh := time.Second
			log.Println("peers unstable, will refresh in", refresh)
			return delayed(t, t.protocol.ClusterChange, refresh)
		}

		return maintainState
	default:
		log.Println("lost leadership: demote to peer")
		return peer{stateMeta: t.stateMeta}.Update(c)
	}
}

// stepdown gives up leadership because this node cannot commit. the followers can't replace a leader
// that still heartbeats. a leadership transfer is tried first, if it can't complete (the leader loop
// is likely the thing that is stuck) the node leaves the protocol, which shuts raft down.
func (t leader) stepdown(c rendezvous, cause error) state {
	log.Println("unable to commit", t.barrierFailures, "consecutive barriers, stepping down:", cause)

	transferred := make(chan error, 1)
	go func() {
		transferred <- t.r.LeadershipTransfer().Error()
	}()

	select {
	case err := <-transferred:
		if err == nil {
			log.Println("leadership transferred")
			return peer{stateMeta: t.stateMeta}.Update(c)
		}
		log.Println("leadership transfer failed, leaving the protocol", err)
	case <-time.After(t.stepdownTimeout):
		log.Println("leadership transfer timed out, leaving the protocol")
	}

	return leave(t.stateMeta)
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

	config := t.r.GetConfiguration()
	if err = config.Error(); err != nil {
		log.Println("failed to retrieve peers", err)
		return true
	}

	peers := config.Configuration().Servers
	unmodifiedPeers := config.Configuration().Servers

	voterCount := voterCount(peers)
	// TODO: temporarily allowing removals at all times.
	// we're hitting deadlocks where we can't remove a peer that rejoined.
	// due to this logic.
	// allowRemoval := voterCount >= 3
	_ = voterCount
	allowRemoval := true

	// we bail out when we fail to add peers because we don't want to remove peers
	// if we failed to add the new peers to the leadership
	for _, peer := range candidates {
		if rs, err = NodeToServer(peer); err != nil {
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
