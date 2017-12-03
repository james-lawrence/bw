package quorum_test

import (
	. "github.com/james-lawrence/bw/agent/quorum"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StateMachine", func() {
	Context("Messages", func() {
		It("should properly apply a command", func() {
			cmd := mustCommand(MessageToCommand(agentutil.LogEvent(agent.NewPeer("foo"), "hello world")))
			obs := &countingEventObserver{}
			fsm := NewStateMachine()
			obsr := fsm.Register(obs)
			defer fsm.Remove(obsr)
			fsm.Apply(&raft.Log{Type: raft.LogCommand, Data: cmd})
			Eventually(func() int { return len(obs.seen) }).Should(Equal(1))
		})
	})
})
