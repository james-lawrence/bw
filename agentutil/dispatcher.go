package agentutil

import (
	"log"
	"sync"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/x/logx"
)

// DiscardDispatcher ...
type DiscardDispatcher struct{}

// Dispatch ...
func (t DiscardDispatcher) Dispatch(ms ...agent.Message) error {
	return nil
}

// LogDispatcher dispatcher that just logs.
type LogDispatcher struct{}

// Dispatch ....
func (t LogDispatcher) Dispatch(ms ...agent.Message) error {
	for _, m := range ms {
		log.Printf("dispatched %#v\n", m)
	}
	return nil
}

// NewBusDispatcher creates a in memory bus for messages.
func NewBusDispatcher(c chan agent.Message) BusDispatcher {
	return BusDispatcher{
		buff: c,
	}
}

// BusDispatcher ...
type BusDispatcher struct {
	buff chan agent.Message
}

// Dispatch ...
func (t BusDispatcher) Dispatch(msgs ...agent.Message) error {
	for _, msg := range msgs {
		t.buff <- msg
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
func (t *Dispatcher) Dispatch(m ...agent.Message) error {
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
