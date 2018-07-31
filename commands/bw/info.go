package main

import (
	"log"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/commands/commandutils"
	"github.com/james-lawrence/bw/ux"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

type agentInfo struct {
	global      *global
	environment string
}

func (t *agentInfo) configure(parent *kingpin.CmdClause) {
	common := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	t.infoCmd(common(parent.Command("all", "retrieve info from all nodes within the cluster").Default()))
}

func (t *agentInfo) infoCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.info)
}

func (t *agentInfo) info(ctx *kingpin.ParseContext) error {
	return t._info()
}

func (t *agentInfo) _info() (err error) {
	var (
		c      clustering.Cluster
		client agent.Client
		d      agent.Dialer
		config agent.ConfigClient
	)
	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}
	log.Println("configuration", spew.Sdump(config))
	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	coptions := []agent.ConnectOption{
		agent.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if client, d, c, err = config.Connect(coptions...); err != nil {
		return err
	}

	logx.MaybeLog(errors.Wrap(client.Close(), "failed to close unused client"))

	cx := cluster.New(local, c)
	err = agentutil.NewClusterOperation(agentutil.Operation(func(c agent.Client) (err error) {
		var (
			info agent.StatusResponse
		)

		if info, err = c.Info(); err != nil {
			return errors.WithStack(err)
		}

		log.Printf("Server: %s:%s - %s\n", info.Peer.Name, info.Peer.Ip, info.Peer.Status)
		log.Println("Previous Deployment: Time                          - DeploymentID               - Stage")
		for _, d := range info.Deployments {
			log.Printf("Previous Deployment: %s - %s - %s", time.Unix(d.Archive.Ts, 0).UTC(), bw.RandomID(d.Archive.DeploymentID), d.Stage)
		}

		return nil
	}))(cx, d)
	logx.MaybeLog(err)

	events := make(chan agent.Message, 100)

	t.global.cleanup.Add(1)
	go ux.Logging(t.global.ctx, t.global.cleanup, events)

	log.Println("awaiting events")
	agentutil.WatchClusterEvents(d, cx, events)
	return nil
}
