package ux

import (
	"container/ring"
	"log"

	"bitbucket.org/jatone/bearded-wookie/agent"
)

func newLBuffer(n int) lbuffer {
	return lbuffer{ring: ring.New(n)}
}

type lbuffer struct {
	ring *ring.Ring
}

func (t lbuffer) Add(m agent.Message) lbuffer {
	t.ring.Value = m
	t.ring = t.ring.Next()
	return t
}

func (t lbuffer) Do(f func(agent.Message)) {
	t.ring.Do(func(x interface{}) {
		if x == nil {
			return
		}

		if m, ok := x.(agent.Message); ok {
			f(m)
			return
		}

		log.Println("type cast failed, ignoring", x)
	})
}
