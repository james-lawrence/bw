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
	gagent "bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"github.com/alecthomas/kingpin"
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
		info   gagent.Status
	)
	defer t.global.shutdown()

	local := cluster.NewLocal(
		gagent.Peer{
			Name: t.node,
			Ip:   t.global.systemIP.String(),
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
		),
	}

	if creds, client, c, err = agent.ConnectLeader(&t.config, coptions...); err != nil {
		return err
	}

	_connector := newConnector(grpc.WithTransportCredentials(creds))
	for _, m := range c.Members() {
		if info, err = _connector.Check2(m); err != nil {
			log.Println("failed to retrieve info for", m.Name, m.Address())
			continue
		}

		log.Printf("Server: %s:%s\n", m.Name, m.Address())
		log.Printf("Status: %s\n", info.Peer.Status.String())
		log.Println("Previous Deployment: Time                          - DeploymentID               - Checksum")
		for _, d := range info.Deployments {
			log.Printf("Previous Deployment: %s - %s - %s", time.Unix(d.Ts, 0).UTC(), bw.RandomID(d.DeploymentID), bw.RandomID(d.Checksum))
		}
	}

	events := make(chan gagent.Message, 100)
	go func() {
		log.Println("awaiting event")
		for m := range events {
			switch m.Type {
			case gagent.Message_PeerEvent:
				p := m.GetPeer()
				log.Printf("%s - %s: %s, %s, %s\n", time.Unix(m.GetTs(), 0).Format(time.Stamp), m.Type, p.Name, p.Ip, p.Status)
			default:
				log.Printf("%s - %s: \n", time.Unix(m.GetTs(), 0).Format(time.Stamp), m.Type)
			}
			log.Println("awaiting event")
		}
	}()

	go func() {
		for _ = range time.Tick(200 * time.Millisecond) {
			client.Dispatch(agentutil.PeerEvent(local.Peer))
		}
	}()
	return client.Watch(events)
}
