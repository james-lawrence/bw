package agent

import (
	"log"

	"bitbucket.org/jatone/bearded-wookie/clustering/raftutil"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type pbObserver struct {
	dst  agent.Leader_WatchServer
	done chan struct{}
}

func (t pbObserver) Receive(messages ...agent.Message) (err error) {
	for _, m := range messages {
		if err = t.dst.Send(&m); err != nil {
			close(t.done)
			return errors.WithStack(err)
		}
	}

	return nil
}

// NewLeader ...
func NewLeader(r *raftutil.Protocol, s Server) Leader {
	return Leader{
		r:        r,
		s:        s,
		EventBus: NewEventBus(),
	}
}

// Leader implements leader functionality.
type Leader struct {
	r        *raftutil.Protocol
	s        Server
	EventBus EventBus
}

// Watch watch for events.
func (t Leader) Watch(_ *agent.WatchRequest, out agent.Leader_WatchServer) (err error) {
	if _, err = t.getLeader(); err != nil {
		return err
	}

	done := make(chan struct{})
	log.Println("event observer: registering")
	o := t.EventBus.Register(pbObserver{dst: out, done: done})
	log.Println("event observer: registered")
	defer t.EventBus.Remove(o)
	<-done

	return nil
}

// Dispatch record deployment events.
func (t Leader) Dispatch(in agent.Leader_DispatchServer) error {
	var (
		err error
		m   *agent.Message
	)

	if _, err = t.getLeader(); err != nil {
		return err
	}

	for m, err = in.Recv(); err == nil; m, err = in.Recv() {
		t.EventBus.Dispatch(*m)
	}

	return nil
}

func (t Leader) getLeader() (*raft.Raft, error) {
	var (
		r *raft.Raft
	)

	if r = t.r.Raft(); r == nil {
		return nil, errors.New("not part of a raft cluster")
	}

	if r.State() != raft.Leader {
		return nil, errors.Errorf("watch can only be executed on the leader agent")
	}

	return r, nil
}
