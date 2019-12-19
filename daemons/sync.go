package daemons

import (
	"time"

	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/timex"
)

// Sync the cluster configuration to the local agent.
func Sync(ctx Context, cx cluster) (err error) {
	dialer := agent.NewDialer(
		agent.DefaultDialerOptions(
			grpc.WithTransportCredentials(ctx.GRPCCreds()),
		)...,
	)

	logx.MaybeLog(_multisync(dialer, cx))
	go timex.Every(time.Hour, func() {
		logx.MaybeLog(_multisync(dialer, cx))
	})

	return nil
}

func _multisync(d dialer, cx cluster) (err error) {
	for i := 0; i < 10; i++ {
		if err = _sync(d, cx); err == nil {
			return nil
		}
	}

	return errorsx.String("failed to sync")
}

func _sync(d dialer, cx cluster) (err error) {
	return nil
	// TODO:
	// var (
	// 	c *grpc.ClientConn
	// )
	//
	// dialer := agent.NewQuorumDialer(d)
	// if c, err = agent.MaybeConn(dialer.Dial(cx)); err != nil {
	// 	return err
	// }
	//
	// quorum.NewConfigurationClient(c)
	// return nil
}
