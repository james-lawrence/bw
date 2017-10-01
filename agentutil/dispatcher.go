package agentutil

import (
	"log"
	"sync"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/x/logx"
	"google.golang.org/grpc"
)

// NewDispatcher create a message dispatcher from the cluster and credentials.
func NewDispatcher(c cluster, creds grpc.DialOption) Dispatcher {
	return Dispatcher{
		cluster: c,
		creds:   creds,
		m:       &sync.Mutex{},
	}
}

// Dispatcher - dispatches messages.
type Dispatcher struct {
	cluster
	c     agent.Client
	creds grpc.DialOption
	m     *sync.Mutex
}

// Dispatch dispatches messages
func (t Dispatcher) Dispatch(m ...agent.Message) error {
	var (
		err error
		c   agent.Client
	)

	if c, err = t.getClient(); err != nil {
		log.Println("-------------- dispatching failed---------------")
		return err
	}

	return logx.MaybeLog(c.Dispatch(m...))
}

func (t Dispatcher) getClient() (c agent.Client, err error) {
	t.m.Lock()
	defer t.m.Unlock()
	if t.c != nil {
		return t.c, nil
	}

	if t.c, err = DialQuorum(t.cluster, t.creds); err != nil {
		return t.c, err
	}

	return t.c, nil
}
