package quorum_test

import (
	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agent/quorum"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("History", func() {
	It("should not record heartbeats", func() {
		local := agent.NewPeer("local")
		h := NewHistory()
		begin := agent.NewDeployCommand(local, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}})

		Expect(h.Decode(TranscoderContext{}, begin)).To(Succeed())
		for i := 0; i < 200; i++ {
			Expect(h.Decode(TranscoderContext{}, agent.NewDeployHeartbeat(local))).To(Succeed())
		}

		snapshot := h.Snapshot()
		Expect(snapshot).To(HaveLen(1))
		Expect(snapshot[0].Id).To(Equal(begin.Id))
	})
})
