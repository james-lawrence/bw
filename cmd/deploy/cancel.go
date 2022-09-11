package deploy

import (
	"log"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/cmd/termui"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/internal/logx"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func Cancel(ctx *Context) (err error) {
	var (
		conn   *grpc.ClientConn
		client agent.DeployClient
		config agent.ConfigClient
		d      dialers.Defaults
		c      clustering.Rendezvous
		ss     notary.Signer
	)

	defer ctx.CancelFunc()

	if config, err = commandutils.LoadConfiguration(ctx.Environment, agent.CCOptionInsecure(ctx.Insecure)); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	events := make(chan *agent.Message, 100)

	local := commandutils.NewClientPeer()

	events <- agentutil.LogEvent(local, "connecting to cluster")
	if d, c, err = daemons.ConnectClientUntilSuccess(ctx.Context, config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	if conn, err = qd.DialContext(ctx.Context); err != nil {
		return err
	}

	go func() {
		<-ctx.Context.Done()
		logx.MaybeLog(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	client = agent.NewDeployConn(conn)

	termui.New(ctx.Context, ctx.CancelFunc, ctx.WaitGroup, qd, local, events)

	events <- agentutil.LogEvent(local, "connected to cluster")
	go func() {
		<-ctx.Context.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	cmd := agentutil.DeployCommandCancel(bw.DisplayName())

	if err = client.Cancel(ctx.Context, &agent.CancelRequest{Initiator: cmd.Initiator}); err != nil {
		return err
	}

	events <- agentutil.LogEvent(local, "deploy cancelled")

	return nil
}
