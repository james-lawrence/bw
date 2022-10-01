package quorum

import (
	"container/ring"
	"io"
	"log"
	"sync"

	"github.com/james-lawrence/bw/agent"
)

// NewHistory tracks the history of a deployment
func NewHistory() History {
	return History{
		lbuffer: newLBuffer(128),
	}
}

// History used to observe messages processed by the state machine.
type History struct {
	lbuffer *lbuffer
}

// Decode consume the messages passing them to observers
func (t History) Decode(ctx TranscoderContext, m *agent.Message) error {
	if m.Hidden {
		return nil
	}

	switch m.Type {
	case agent.Message_DeployCommandEvent:
		switch m.GetDeployCommand().Command {
		case agent.DeployCommand_Begin:
			t.lbuffer.Reset()
			t.lbuffer.Add(m)
		default:
			t.lbuffer.Add(m)
		}
	default:
		t.lbuffer.Add(m)
	}

	return nil
}

func (t History) Snapshot() []*agent.Message {
	return t.lbuffer.Snapshot()
}

// Encode satisfy the transcoder interface. does nothing.
func (t History) Encode(dst io.Writer) (err error) {
	return nil
}

func newLBuffer(n int) *lbuffer {
	return &lbuffer{ring: ring.New(n)}
}

type lbuffer struct {
	m    sync.RWMutex
	ring *ring.Ring
}

func (t *lbuffer) Reset() *lbuffer {
	t.m.Lock()
	defer t.m.Unlock()
	t.ring = ring.New(t.ring.Len())
	return t
}

func (t *lbuffer) Add(msgs ...*agent.Message) *lbuffer {
	t.m.Lock()
	defer t.m.Unlock()
	for _, m := range msgs {
		t.ring.Value = m
		t.ring = t.ring.Next()
	}

	return t
}

func (t *lbuffer) Snapshot(msgs ...*agent.Message) []*agent.Message {
	t.m.RLock()
	defer t.m.RUnlock()

	history := make([]*agent.Message, 0, 100)
	t.do(func(m *agent.Message) {
		history = append(history, m)
	})

	return history
}

func (t *lbuffer) do(f func(*agent.Message)) {
	t.ring.Do(func(x interface{}) {
		if x == nil {
			return
		}

		if m, ok := x.(*agent.Message); ok {
			f(m)
			return
		}

		log.Println("type cast failed, ignoring", x)
	})
}
