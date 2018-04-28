package observers

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

// New observer
func New(b chan agent.Message) (s *grpc.Server) {
	s = grpc.NewServer()

	o := Observer{
		bus: b,
	}

	agent.RegisterObserverServer(s, o)

	return s
}

// Observer observes events of the cluster.
type Observer struct {
	bus chan agent.Message
}

// Dispatch receives events from the cluster.
func (t Observer) Dispatch(ctx context.Context, in *agent.DispatchRequest) (out *agent.DispatchResponse, err error) {
	messages := agent.MessagesFromPtr(in.Messages...)
	for _, m := range messages {
		// log.Println("received", len(t.bus), cap(t.bus))
		select {
		case t.bus <- m:
		case <-ctx.Done():
			return &agent.DispatchResponse{}, ctx.Err()
		}
		// log.Println("buffered", len(t.bus), cap(t.bus))
	}

	return &agent.DispatchResponse{}, err
}
