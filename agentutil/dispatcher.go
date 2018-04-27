package agentutil

import (
	"context"
	"log"
	"sync"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

// DiscardDispatcher ...
type DiscardDispatcher struct{}

// Dispatch ...
func (t DiscardDispatcher) Dispatch(_ context.Context, ms ...agent.Message) error {
	return nil
}

// LogDispatcher dispatcher that just logs.
type LogDispatcher struct{}

// Dispatch ....
func (t LogDispatcher) Dispatch(_ context.Context, ms ...agent.Message) error {
	for _, m := range ms {
		log.Printf("dispatched %#v\n", m)
	}
	return nil
}

// NewDispatcher create a message dispatcher from the cluster and credentials.
func NewDispatcher(c cluster, d agent.QuorumDialer) *Dispatcher {
	return &Dispatcher{
		cluster: c,
		dialer:  d,
		m:       &sync.Mutex{},
	}
}

// Dispatcher - dispatches messages.
type Dispatcher struct {
	cluster
	dialer agent.QuorumDialer
	c      agent.Client
	m      *sync.Mutex
}

// Dispatch dispatches messages
func (t *Dispatcher) Dispatch(ctx context.Context, m ...agent.Message) (err error) {
	var (
		c agent.Client
	)

	if c, err = t.getClient(); err != nil {
		log.Println("-------------- dispatching failed---------------")
		return err
	}

	return logx.MaybeLog(t.dropClient(c.Dispatch(ctx, m...)))
}

func (t *Dispatcher) getClient() (c agent.Client, err error) {
	t.m.Lock()
	defer t.m.Unlock()
	if t.c != nil {
		return t.c, nil
	}

	if t.c, err = t.dialer.Dial(t.cluster); err != nil {
		return t.c, err
	}

	return t.c, nil
}

func (t *Dispatcher) dropClient(err error) error {
	if err == nil {
		return err
	}

	t.m.Lock()
	t.c = nil
	t.m.Unlock()

	return errors.Wrap(err, "dropped client due to error")
}
