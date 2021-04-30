package observers_test

import (
	"context"
	"time"

	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agent/observers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/x/testingx"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Observers", func() {
	It("should be able to receive messages", func() {
		dst := make(chan *agent.Message, 10)
		d, srv := testingx.NewGRPCServer2(func(s *grpc.Server) {
			New(dst).Bind(s)
		})
		defer testingx.GRPCCleanup(nil, srv)

		conn, err := d.DialContext(context.Background(), grpc.WithBlock())
		Expect(err).To(Succeed())

		obsc := NewConn(conn)
		Expect(err).To(Succeed())
		Expect(
			obsc.Dispatch(
				context.Background(),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 0"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 1"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 2"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 3"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 4"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 5"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 6"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 7"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 8"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 9"),
			),
		).To(Succeed())

		// 2nd batch, should time out due to a full buffer.
		ctx, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		Expect(
			obsc.Dispatch(
				ctx,
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 10"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 11"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 12"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 13"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 14"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 15"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 16"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 17"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 18"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 19"),
			),
		).To(HaveOccurred())

		Expect(len(dst)).To(Equal(10))
	})
})
