package agent

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/x/debugx"
)

// NewEventBusDefault ...
func NewEventBusDefault() EventBus {
	return NewEventBus(make(chan []Message, 1000))
}

// NewEventBus event bus.
func NewEventBus(c chan []Message) EventBus {
	e := EventBus{
		buffer:    c,
		m:         &sync.RWMutex{},
		observers: map[int64]Observer{},
		serial:    new(int64),
	}

	go e.background()

	return e
}

// Observer ...
type Observer interface {
	Receive(context.Context, ...Message) error
}

// EventBusObserver used to unregister an observer.
type EventBusObserver struct {
	id int64
}

// EventBus - message bus for events on the leader node.
type EventBus struct {
	buffer    chan []Message
	serial    *int64
	m         *sync.RWMutex
	observers map[int64]Observer
}

func (t EventBus) background() {
	for m := range t.buffer {
		t.m.RLock()
		cpy := make([]Observer, 0, len(t.observers))
		for _, obs := range t.observers {
			cpy = append(cpy, obs)
		}
		t.m.RUnlock()

		debugx.Printf("receiving messages(%d), observers(%d)\n", len(m), len(t.observers))
		for _, o := range cpy {
			c, cf := context.WithTimeout(context.Background(), time.Minute)
			if err := o.Receive(c, m...); err != nil {
				cf()
				log.Printf("failed to receive messages - %T %+v\n", errors.Cause(err), err)
				continue
			}
			cf()
			debugx.Println("delivered messages")
		}
	}
}

// Dispatch ...
func (t EventBus) Dispatch(ctx context.Context, messages ...Message) error {
	log.Println("dispatching messages", len(messages))
	select {
	case <-ctx.Done():
		log.Println("done")
		return ctx.Err()
	case t.buffer <- messages:
		log.Println("dispatched messages", len(messages))
	}

	return nil
}

// Register ...
func (t EventBus) Register(o Observer) EventBusObserver {
	t.m.Lock()
	defer t.m.Unlock()
	obs := EventBusObserver{
		id: atomic.AddInt64(t.serial, 1),
	}

	debugx.Println("registering", obs.id, len(t.observers))
	t.observers[obs.id] = o

	return obs
}

// Remove ...
func (t EventBus) Remove(e EventBusObserver) {
	t.m.Lock()
	defer t.m.Unlock()
	debugx.Println("removing observer", e.id, len(t.observers))
	delete(t.observers, e.id)
}
