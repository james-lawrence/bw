package quorum

import (
	"io"
	"log"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering/raftutil"
)

func NewLeadershipTransfer(c cluster, rp raftutil.Protocol) *LeadershipTransfer {
	return &LeadershipTransfer{
		c:  c,
		rp: rp,
	}
}

// LeadershipTransfer used to transfer leadership after a deploy
type LeadershipTransfer struct {
	c       cluster
	rp      raftutil.Protocol
	current *raft.Raft
}

func (t *LeadershipTransfer) NewRaftObserver() *raft.Observer {
	return raft.NewObserver(nil, false, func(o *raft.Observation) bool {
		// track the latest raft instance
		t.current = o.Raft
		return false
	})
}

// Decode consume the messages waiting for a deployment to finish
func (t *LeadershipTransfer) Decode(ctx TranscoderContext, m *agent.Message) error {
	if t.current == nil {
		return nil
	}

	// only consider deploy complete or deploy failed
	// for leadership transfers
	switch evt := m.Event.(type) {
	case *agent.Message_DeployCommand:
		if evt.DeployCommand == nil {
			return nil
		}

		switch evt.DeployCommand.Command {
		case agent.DeployCommand_Done, agent.DeployCommand_Failed:
		default:
			return nil
		}
	default:
		return nil
	}

	// if current isn't the leader then ignore.
	if t.current.State() != raft.Leader {
		return nil
	}

	// if the node is still a member of quorum then ignore.
	if !t.rp.MaybeLeave(t.c) {
		return nil
	}

	log.Println("leadership transfer initiated")
	defer log.Println("leadership transfer completed")

	if err := t.current.LeadershipTransfer().Error(); err != nil {
		log.Println("unable to transfer leadership", err)
	}

	t.rp.ClusterChange.Broadcast()

	return nil
}

// Encode satisfy the transcoder interface. does nothing.
func (t *LeadershipTransfer) Encode(dst io.Writer) (err error) {
	return nil
}
