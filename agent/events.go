package agent

import (
	"log"
	"sync"
	"sync/atomic"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

// NewEventBus event bus.
func NewEventBus() EventBus {
	e := EventBus{
		buffer:    make(chan []agent.Message, 1000),
		m:         &sync.RWMutex{},
		observers: map[int64]eventBusObserver{},
		serial:    new(int64),
	}

	go e.background()

	return e
}

type eventBusObserver interface {
	Receive(...agent.Message) error
}

// EventBusObserver used to unregister an observer.
type EventBusObserver struct {
	id int64
}

// EventBus - message bus for events on the leader node.
type EventBus struct {
	buffer    chan []agent.Message
	serial    *int64
	m         *sync.RWMutex
	observers map[int64]eventBusObserver
}

func (t EventBus) background() {
	for m := range t.buffer {
		t.m.RLock()
		obs := t.observers
		t.m.RUnlock()
		log.Printf("receiving messages(%d), observers(%d)\n", len(m), len(obs))
		for id, o := range obs {
			if err := o.Receive(m...); err != nil {
				log.Println("failed to receive messages", id, err)
			}
		}
	}
}

// Dispatch ...
func (t EventBus) Dispatch(messages ...agent.Message) {
	log.Println("dispatching messages", len(messages))
	t.buffer <- messages
}

// Register ...
func (t EventBus) Register(o eventBusObserver) EventBusObserver {
	t.m.Lock()
	defer t.m.Unlock()
	obs := EventBusObserver{
		id: atomic.AddInt64(t.serial, 1),
	}

	log.Println("registering", obs.id)
	t.observers[obs.id] = o

	return obs
}

// Remove ...
func (t EventBus) Remove(e EventBusObserver) {
	t.m.Lock()
	defer t.m.Unlock()

	delete(t.observers, e.id)
}
