package agent

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/x/debugx"
)

// NewEventBus event bus.
func NewEventBus() EventBus {
	e := EventBus{
		buffer:    make(chan []Message, 1000),
		m:         &sync.RWMutex{},
		observers: map[int64]Observer{},
		serial:    new(int64),
	}

	go e.background()

	return e
}

// Observer ...
type Observer interface {
	Receive(...Message) error
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
		debugx.Printf("receiving messages(%d), observers(%d)\n", len(m), len(t.observers))
		for id, o := range t.observers {
			if err := o.Receive(m...); err != nil {
				log.Printf("failed to receive messages %d - %T %+v\n", id, errors.Cause(err), err)
				continue
			}

			debugx.Println("delivered messages", id)
		}
		t.m.RUnlock()
	}
}

// Dispatch ...
func (t EventBus) Dispatch(messages ...Message) {
	debugx.Println("dispatching messages", len(messages))
	t.buffer <- messages
	debugx.Println("dispatched messages", len(messages))
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
