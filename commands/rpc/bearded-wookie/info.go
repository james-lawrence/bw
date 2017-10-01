package main

import (
	"log"
	"path/filepath"
	"time"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/ux"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"
	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type agentInfo struct {
	config      agent.ConfigClient
	global      *global
	node        string
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
		creds  credentials.TransportCredentials
		client agent.Client
	)
	defer t.global.shutdown()

	local := cluster.NewLocal(
		agent.Peer{
			Name: t.node,
			Ip:   systemx.HostnameOrLocalhost(),
		},
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Deploy)),
	)

	coptions := []agent.ConnectOption{
		agent.ConnectOptionConfigPath(filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), t.environment)),
		agent.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(eventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if creds, client, c, err = agent.ConnectLeader(&t.config, coptions...); err != nil {
		return err
	}

	cx := cluster.New(local, c)
	agentutil.NewClusterOperation(agentutil.Operation(func(c agent.Client) (err error) {
		var (
			info agent.Status
		)
		if info, err = c.Info(); err != nil {
			return errors.WithStack(err)
		}

		log.Printf("Server: %s:%s\n", info.Peer.Name, info.Peer.Ip)
		log.Printf("Status: %s\n", info.Peer.Status.String())
		log.Println("Previous Deployment: Time                          - DeploymentID               - Checksum")
		for _, d := range info.Deployments {
			log.Printf("Previous Deployment: %s - %s - %s", time.Unix(d.Ts, 0).UTC(), bw.RandomID(d.DeploymentID), bw.RandomID(d.Checksum))
		}

		return nil
	}))(cx, grpc.WithTransportCredentials(creds))

	events := make(chan agent.Message, 100)
	go ux.Logging(events)

	log.Println("awaiting events")
	return client.Watch(events)
}
