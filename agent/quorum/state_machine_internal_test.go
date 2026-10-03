package quorum

import (
	"context"
	"sync"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// dispatcher that fails the first n dispatches.
type flakyDispatcher struct {
	m         sync.Mutex
	failures  int
	attempts  int
	delivered []*agent.Message
}

func (t *flakyDispatcher) Dispatch(_ context.Context, ms ...*agent.Message) error {
	t.m.Lock()
	defer t.m.Unlock()

	t.attempts++
	if t.attempts <= t.failures {
		return status.Error(codes.Unavailable, "connection is unavailable")
	}

	t.delivered = append(t.delivered, ms...)
	return nil
}

var _ = Describe("dispatchDeployResult", func() {
	It("should retry until the deploy result is delivered", func() {
		local := agent.NewPeer("local")
		d := &flakyDispatcher{failures: 1}
		dcmd := agent.DeployCommandDone("tester")

		Expect(dispatchDeployResult(context.Background(), d, local, dcmd)).To(Succeed())
		Expect(d.attempts).To(Equal(2))
		Expect(d.delivered).To(HaveLen(1))
		Expect(d.delivered[0].GetDeployCommand().Command).To(Equal(agent.DeployCommand_Done))
	})
})
