package agent_test

import (
	. "github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"

	"github.com/hashicorp/raft"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fsm", func() {
	Context("Messages", func() {
		It("should properly apply a command", func() {
			cmd := mustCommand(MessageToCommand(agentutil.LogEvent(LocalPeer("foo"), "hello world")))
			obs := &countingEventObserver{}
			fsm := NewQuorumFSM()
			obsr := fsm.Register(obs)
			defer fsm.Remove(obsr)
			fsm.Apply(&raft.Log{Type: raft.LogCommand, Data: cmd})
			Eventually(func() int { return len(obs.seen) }).Should(Equal(1))
		})
	})
})
