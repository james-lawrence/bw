package bootstrap

import (
	"context"
	"net"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Run a bootstrap socket
func Run(ctx context.Context, socket string, o agent.BootstrapServer, options ...grpc.ServerOption) error {
	var (
		err error
		l   net.Listener
	)

	if len(socket) == 0 {
		return errors.New("invalid socket")
	}

	if l, err = net.Listen("unix", socket); err != nil {
		return err
	}

	s := grpc.NewServer(options...)
	agent.RegisterBootstrapServer(s, o)

	go s.Serve(l)
	go func() {
		<-ctx.Done()
		logx.MaybeLog(errors.Wrap(l.Close(), "during bootstrap socket shutdown"))
	}()

	return nil
}