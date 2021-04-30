package observers

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

// New observer
func New(b chan *agent.Message) Observer {
	return Observer{
		bus: b,
	}
}

// Observer observes events of the cluster.
type Observer struct {
	agent.UnimplementedObserverServer
	bus chan *agent.Message
}

func (t Observer) Bind(s *grpc.Server) {
	agent.RegisterObserverServer(s, t)
}

// Dispatch receives events from the cluster.
func (t Observer) Dispatch(ctx context.Context, in *agent.DispatchRequest) (out *agent.DispatchResponse, err error) {
	for idx, m := range in.Messages {
		// log.Println("received", len(t.bus), cap(t.bus))
		select {
		case t.bus <- m:
		case <-ctx.Done():
			log.Println("dropping messages on floor", len(in.Messages)-idx, ctx.Err())
			return &agent.DispatchResponse{}, ctx.Err()
		}
		// log.Println("buffered", len(t.bus), cap(t.bus))
	}

	return &agent.DispatchResponse{}, err
}
