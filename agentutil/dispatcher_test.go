package agentutil_test

import (
	"context"
	"sync/atomic"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	. "github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/testingx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func dispatchLog(req *agent.DispatchRequest) string {
	if len(req.Messages) == 0 {
		return ""
	}

	return req.Messages[0].GetLog().GetLog()
}

// quorum server that blocks "slow" messages until released and rejects "fail" messages.
type blockingQuorum struct {
	agent.UnimplementedQuorumServer
	received chan string
	release  chan struct{}
}

func (t blockingQuorum) Dispatch(ctx context.Context, req *agent.DispatchRequest) (*agent.DispatchResponse, error) {
	msg := dispatchLog(req)
	t.received <- msg

	switch msg {
	case "slow":
		select {
		case <-t.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case "fail", "late":
		return nil, status.Error(codes.Unavailable, "server unavailable")
	}

	return &agent.DispatchResponse{}, nil
}

type countingDialer struct {
	dialers.Direct
	dials *int64
}

func (t countingDialer) DialContext(ctx context.Context, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	atomic.AddInt64(t.dials, 1)
	return t.Direct.DialContext(ctx, options...)
}

var _ = Describe("Dispatcher", func() {
	var (
		local  = agent.NewPeer("local")
		srv    blockingQuorum
		server *grpc.Server
		dialer countingDialer
		// client side hold on the "late" message's response, simulating an rpc whose failure
		// is observed after the dispatcher has already moved on to a new connection.
		lateheld    chan struct{}
		laterelease chan struct{}
	)

	BeforeEach(func() {
		srv = blockingQuorum{
			received: make(chan string, 10),
			release:  make(chan struct{}),
		}
		lateheld = make(chan struct{})
		laterelease = make(chan struct{})

		holdlate := grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if dr, ok := req.(*agent.DispatchRequest); ok && dispatchLog(dr) == "late" {
				close(lateheld)
				<-laterelease
			}
			return err
		})

		var d dialers.Direct
		d, server = testingx.NewGRPCServer2(func(s *grpc.Server) {
			agent.RegisterQuorumServer(s, srv)
		}, holdlate)
		dialer = countingDialer{Direct: d, dials: new(int64)}
	})

	AfterEach(func() {
		server.Stop()
	})

	It("should not close the shared connection when a caller's context is cancelled", func() {
		d := NewDispatcher(dialer)

		inflight := make(chan error, 1)
		go func() {
			inflight <- d.Dispatch(context.Background(), agent.LogEvent(local, "slow"))
		}()
		Eventually(srv.received).Should(Receive(Equal("slow")))

		// e.g. a heartbeat whose context was cancelled when the deploy completed.
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		Expect(d.Dispatch(cancelled, agent.LogEvent(local, "heartbeat"))).To(HaveOccurred())

		close(srv.release)
		Eventually(inflight).Should(Receive(Succeed()))
		Expect(atomic.LoadInt64(dialer.dials)).To(Equal(int64(1)))
	})

	It("should not drop a newer connection when a stale rpc fails", func() {
		d := NewDispatcher(dialer)

		Expect(d.Dispatch(context.Background(), agent.LogEvent(local, "ok"))).To(Succeed())
		Expect(atomic.LoadInt64(dialer.dials)).To(Equal(int64(1)))

		// rpc on connection #1 that fails, but whose failure is observed late.
		late := make(chan error, 1)
		go func() {
			late <- d.Dispatch(context.Background(), agent.LogEvent(local, "late"))
		}()
		Eventually(lateheld).Should(BeClosed())

		// connection #1 is dropped and connection #2 is established.
		Expect(d.Dispatch(context.Background(), agent.LogEvent(local, "fail"))).To(HaveOccurred())
		Expect(d.Dispatch(context.Background(), agent.LogEvent(local, "ok"))).To(Succeed())
		Expect(atomic.LoadInt64(dialer.dials)).To(Equal(int64(2)))

		// the stale failure from connection #1 arrives.
		close(laterelease)
		Eventually(late).Should(Receive(HaveOccurred()))

		// connection #2 should still be in use.
		Expect(d.Dispatch(context.Background(), agent.LogEvent(local, "ok"))).To(Succeed())
		Expect(atomic.LoadInt64(dialer.dials)).To(Equal(int64(2)))
	})
})
