package observers

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewConn observing connection.
func NewConn(c *grpc.ClientConn) Conn {
	return Conn{conn: c}
}

// Conn connection to the observer
type Conn struct {
	conn *grpc.ClientConn
}

// Dispatch the messages.
func (t Conn) Dispatch(ctx context.Context, messages ...*agent.Message) (err error) {
	var (
		dispatch = agent.DispatchRequest{
			Messages: messages,
		}
	)

	rpc := agent.NewObserverClient(t.conn)
	if _, err = rpc.Dispatch(ctx, &dispatch); err != nil {
		return errorsx.MaybeLog(errors.Wrap(err, "failed to dispatch messages"))
	}

	return nil
}
