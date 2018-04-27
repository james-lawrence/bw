package observers

import (
	"context"
	"net"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewDialer connect to a unix domain socket.
func NewDialer(ctx context.Context, uds string, options ...grpc.DialOption) (c Conn, err error) {
	var (
		conn *grpc.ClientConn
	)

	options = append(options, grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}))

	if conn, err = grpc.DialContext(ctx, uds, options...); err != nil {
		return c, err
	}

	return Conn{conn: conn}, nil
}

// Conn connection to the observer
type Conn struct {
	conn *grpc.ClientConn
}

// Dispatch the messages.
func (t Conn) Dispatch(ctx context.Context, messages ...agent.Message) (err error) {
	var (
		dispatch = agent.DispatchRequest{
			Messages: agent.MessagesToPtr(messages...),
		}
	)

	rpc := agent.NewObserverClient(t.conn)
	if _, err = rpc.Dispatch(ctx, &dispatch); err != nil {
		return errors.Wrap(err, "failed to dispatch messages")
	}
	return nil
}
