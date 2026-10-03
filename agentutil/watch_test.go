package agentutil_test

import (
	"context"
	"errors"
	"time"

	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agentutil"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type failingDialer struct{}

func (failingDialer) DialContext(ctx context.Context, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	return nil, errors.New("unable to dial")
}

var _ = Describe("WatchEvents", func() {
	It("should return once the context is cancelled even if events are no longer consumed", func() {
		ctx, cancel := context.WithCancel(context.Background())
		// nothing reads from events, e.g. the ux has already exited.
		events := make(chan *agent.Message)
		returned := make(chan struct{})

		go func() {
			defer close(returned)
			WatchEvents(ctx, agent.NewPeer("local"), failingDialer{}, events)
		}()

		// give the watcher time to block delivering the dial failure.
		time.Sleep(50 * time.Millisecond)
		cancel()

		Eventually(returned, time.Second).Should(BeClosed())
	})
})
