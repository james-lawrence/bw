package agentutil_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	. "github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/testingx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/require"
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
	case "busy":
		return nil, status.Error(codes.ResourceExhausted, "back off")
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

type dispatcherFixture struct {
	local  *agent.Peer
	srv    blockingQuorum
	server *grpc.Server
	dialer countingDialer
	// client side hold on the "late" message's response, simulating an rpc whose failure
	// is observed after the dispatcher has already moved on to a new connection.
	lateheld    chan struct{}
	laterelease chan struct{}
}

func newDispatcherFixture(t *testing.T) *dispatcherFixture {
	t.Helper()
	f := &dispatcherFixture{
		local: agent.NewPeer("local"),
		srv: blockingQuorum{
			received: make(chan string, 10),
			release:  make(chan struct{}),
		},
		lateheld:    make(chan struct{}),
		laterelease: make(chan struct{}),
	}

	holdlate := grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if dr, ok := req.(*agent.DispatchRequest); ok && dispatchLog(dr) == "late" {
			close(f.lateheld)
			<-f.laterelease
		}
		return err
	})

	var d dialers.Direct
	d, f.server = testingx.NewGRPCServer2(t, func(s *grpc.Server) {
		agent.RegisterQuorumServer(s, f.srv)
	}, holdlate)
	f.dialer = countingDialer{Direct: d, dials: new(int64)}
	t.Cleanup(f.server.Stop)

	return f
}

func TestDispatcherShouldNotCloseTheSharedConnectionWhenACallersContextIsCancelled(t *testing.T) {
	f := newDispatcherFixture(t)
	d := NewDispatcher(f.dialer)

	inflight := make(chan error, 1)
	go func() {
		inflight <- d.Dispatch(context.Background(), agent.LogInfo(f.local, "slow"))
	}()
	select {
	case msg := <-f.srv.received:
		require.Equal(t, "slow", msg)
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for the slow message")
	}

	// e.g. a heartbeat whose context was cancelled when the deploy completed.
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, d.Dispatch(cancelled, agent.LogInfo(f.local, "heartbeat")))

	close(f.srv.release)
	select {
	case err := <-inflight:
		require.NoError(t, err)
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for the in flight dispatch")
	}
	require.Equal(t, int64(1), atomic.LoadInt64(f.dialer.dials))
}

func TestDispatcherShouldNotDropANewerConnectionWhenAStaleRPCFails(t *testing.T) {
	f := newDispatcherFixture(t)
	d := NewDispatcher(f.dialer)

	require.NoError(t, d.Dispatch(context.Background(), agent.LogInfo(f.local, "ok")))
	require.Equal(t, int64(1), atomic.LoadInt64(f.dialer.dials))

	// rpc on connection #1 that fails, but whose failure is observed late.
	late := make(chan error, 1)
	go func() {
		late <- d.Dispatch(context.Background(), agent.LogInfo(f.local, "late"))
	}()
	select {
	case <-f.lateheld:
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for the late rpc to be held")
	}

	// connection #1 is dropped and connection #2 is established.
	require.Error(t, d.Dispatch(context.Background(), agent.LogInfo(f.local, "fail")))
	require.NoError(t, d.Dispatch(context.Background(), agent.LogInfo(f.local, "ok")))
	require.Equal(t, int64(2), atomic.LoadInt64(f.dialer.dials))

	// the stale failure from connection #1 arrives.
	close(f.laterelease)
	select {
	case err := <-late:
		require.Error(t, err)
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for the stale failure")
	}

	// connection #2 should still be in use.
	require.NoError(t, d.Dispatch(context.Background(), agent.LogInfo(f.local, "ok")))
	require.Equal(t, int64(2), atomic.LoadInt64(f.dialer.dials))
}

func TestDispatcherShouldKeepTheConnectionWhenTheServerAsksToBackOff(t *testing.T) {
	f := newDispatcherFixture(t)
	d := NewDispatcher(f.dialer)

	err := d.Dispatch(context.Background(), agent.LogInfo(f.local, "busy"))
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.NoError(t, d.Dispatch(context.Background(), agent.LogInfo(f.local, "ok")))
	require.EqualValues(t, 1, atomic.LoadInt64(f.dialer.dials), "backing off isn't a connection failure, the connection is reused")
}
