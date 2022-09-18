package observers_test

import (
	"context"
	"time"

	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agent/observers"
	"github.com/james-lawrence/bw/internal/testingx"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo/v2"
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
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 0"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 1"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 2"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 3"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 4"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 5"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 6"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 7"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 8"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 9"),
			),
		).To(Succeed())

		// 2nd batch, should time out due to a full buffer.
		ctx, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		Expect(
			obsc.Dispatch(
				ctx,
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 10"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 11"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 12"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 13"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 14"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 15"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 16"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 17"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 18"),
				agent.LogEvent(agent.NewPeer("peer1"), "hello world 19"),
			),
		).To(HaveOccurred())

		Expect(len(dst)).To(Equal(10))
	})
})
