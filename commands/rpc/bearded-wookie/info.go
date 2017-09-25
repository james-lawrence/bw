package main

import (
	"log"
	"net"
	"path/filepath"
	"time"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	gagent "bitbucket.org/jatone/bearded-wookie/deployment/agent"
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
		info   gagent.Status
		port   string
	)
	defer t.global.shutdown()

	coptions := []agent.ConnectOption{
		agent.ConnectOptionConfigPath(filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), t.environment)),
		agent.ConnectOptionClustering(
			clustering.OptionNodeID(t.node),
			clustering.OptionBindAddress(t.global.systemIP.String()),
			clustering.OptionEventDelegate(eventHandler{}),
		),
	}

	if creds, client, c, err = agent.ConnectClient(&t.config, coptions...); err != nil {
		return err
	}

	if err = client.Close(); err != nil {
		log.Println("failed to close client")
	}

	if _, port, err = net.SplitHostPort(t.config.Address); err != nil {
		return errors.Wrapf(err, "malformed address in configuration: %s", t.config.Address)
	}

	_connector := newConnector(port, grpc.WithTransportCredentials(creds))
	for _, m := range c.Members() {
		if info, err = _connector.Check2(m); err != nil {
			log.Println("failed to retrieve info for", m.Name, m.Address())
		}
		log.Printf("Server: %s:%s\n", m.Name, m.Address())
		log.Printf("Status: %s\n", info.Peer.Status.String())
		log.Println("Previous Deployment: Time                          - DeploymentID               - Checksum")
		for _, d := range info.Deployments {
			log.Printf("Previous Deployment: %s - %s - %s", time.Unix(d.Ts, 0).UTC(), bw.RandomID(d.DeploymentID), bw.RandomID(d.Checksum))
		}
	}

	events := make(chan gagent.Message, 100)
	func() {
		log.Println("awaiting event")
		for m := range events {
			switch m.Type {
			default:
				log.Printf("%s - %s: \n", time.Unix(m.GetTs(), 0).Format(time.Stamp), m.Type)
			}
			log.Println("awaiting event")
		}
	}()

	return client.Watch(events)
}
