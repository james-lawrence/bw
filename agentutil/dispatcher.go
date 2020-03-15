package agentutil

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

const (
	dispatchTimeout = 10 * time.Second
)

type dispatcher interface {
	Dispatch(context.Context, ...agent.Message) error
}

// Dispatch messages using the provided dispatcher will log and return the error,
// if any, that occurs.
func Dispatch(d dispatcher, m ...agent.Message) error {
	return logx.MaybeLog(_dispatch(context.Background(), d, dispatchTimeout, m...))
}

// ReliableDispatch repeatedly attempts to deliver messages using the provided
// context and dispatcher until the context is cancelled.
func ReliableDispatch(ctx context.Context, d dispatcher, m ...agent.Message) (err error) {
	bs := backoff.Maximum(10*time.Second, backoff.Exponential(200*time.Millisecond))
	for i := 0; ; i++ {
		if err = _dispatch(ctx, d, dispatchTimeout, m...); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			s := bs.Backoff(i)
			log.Output(2, fmt.Sprintf("dispatch attempt %T - %d failed, retrying in %v: %s\n", d, i, s, err))
			time.Sleep(s)
		}
	}
}

func _dispatch(ctx context.Context, d dispatcher, timeout time.Duration, m ...agent.Message) error {
	ctx, done := context.WithTimeout(ctx, timeout)
	defer done()

	return d.Dispatch(ctx, m...)
}

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

	return logx.MaybeLog(t.dropClient(c, c.Dispatch(ctx, m...)))
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

func (t *Dispatcher) dropClient(bad agent.Client, err error) error {
	if err == nil {
		return err
	}

	t.m.Lock()
	logx.MaybeLog(errors.Wrap(bad.Close(), "failed to cleanup client"))
	t.c = nil
	t.m.Unlock()

	return errors.Wrap(err, "dropped client due to error")
}
