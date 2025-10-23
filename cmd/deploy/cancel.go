package deploy

import (
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/cmd/termui"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/vcsinfo"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func Cancel(gctx *Context) (err error) {
	var (
		conn   *grpc.ClientConn
		client agent.DeployClient
		config agent.ConfigClient
		d      dialers.Defaults
		c      clustering.Rendezvous
		ss     notary.Signer
	)

	defer gctx.CancelFunc()

	if config, err = commandutils.LoadConfiguration(gctx.Context, gctx.Environment, agent.CCOptionInsecure(gctx.Insecure)); err != nil {
		return err
	}

	displayname := vcsinfo.CurrentUserDisplay(config.WorkDir())

	if ss, err = notary.NewAutoSigner(displayname); err != nil {
		return err
	}

	events := make(chan *agent.Message, 100)

	local := commandutils.NewClientPeer()

	events <- agent.LogEvent(local, "connecting to cluster")
	if d, c, err = daemons.ConnectClientUntilSuccess(gctx.Context, config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	if conn, err = qd.DialContext(gctx.Context); err != nil {
		return err
	}

	go func() {
		<-gctx.Context.Done()
		errorsx.Log(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	client = agent.NewDeployConn(conn)
	go func() {
		<-gctx.Context.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	termui.NewFromClientConfig(gctx.Context, config, qd, local, events)
	events <- agent.LogEvent(local, "connected to cluster")

	cmd := agent.DeployCommandCancel(displayname)

	if err = client.Cancel(gctx.Context, &agent.CancelRequest{Initiator: cmd.Initiator}); err != nil {
		return err
	}

	events <- agent.LogEvent(local, "deploy cancelled")

	return nil
}
