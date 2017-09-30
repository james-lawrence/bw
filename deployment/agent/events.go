package agent

import (
	"log"
	"sync"
	"sync/atomic"
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
func (t EventBus) Dispatch(messages ...Message) {
	log.Println("dispatching messages", len(messages))
	t.buffer <- messages
}

// Register ...
func (t EventBus) Register(o Observer) EventBusObserver {
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
