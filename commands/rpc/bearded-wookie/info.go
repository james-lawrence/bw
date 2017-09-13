package main

import (
	"log"
	"net"
	"path/filepath"

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
		c           clustering.Cluster
		creds       credentials.TransportCredentials
		coordinator agent.Client
		info        gagent.AgentInfo
		port        string
	)

	if err = bw.ExpandAndDecodeFile(filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), t.environment), &t.config); err != nil {
		return err
	}

	defaults := []clustering.Option{
		clustering.OptionNodeID(t.global.node),
		clustering.OptionBindAddress(t.global.systemIP.String()),
		clustering.OptionEventDelegate(eventHandler{}),
	}

	if creds, coordinator, c, err = t.config.Connect(defaults, []clustering.BootstrapOption{}); err != nil {
		return err
	}

	if err = coordinator.Close(); err != nil {
		log.Println("failed to close coordinator")
	}

	if _, port, err = net.SplitHostPort(t.config.Address); err != nil {
		return errors.Wrap(err, "malformed address in configuration")
	}

	_connector := newConnector(port, grpc.WithTransportCredentials(creds))
	for _, m := range c.Members() {
		if info, err = _connector.Check2(m); err != nil {
			log.Println("failed to retrieve info for", m.Name, m.Address())
		}
		log.Printf("Server: %s:%s\n", m.Name, m.Address())
		log.Printf("Status: %s\n", info.Status.String())
		log.Println("Previous Deployment: DeploymentID               - Checksum")
		for _, d := range info.Deployments {
			log.Printf("Previous Deployment: %s - %s", bw.RandomID(d.DeploymentID), bw.RandomID(d.Checksum))
		}
	}

	return nil
}
