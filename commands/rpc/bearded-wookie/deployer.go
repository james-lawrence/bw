package main

import (
	"log"
	"net"
	"net/rpc"
	"os"

	cp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/ux"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/memberlist"
)

type deployer struct {
	*global
}

func (t *deployer) configure(parent *kingpin.CmdClause) {
	t.global.cluster.configure(parent)
	parent.Command("all", "deploy to all nodes within the cluster").Default().Action(t.Deploy)
}

func (t *deployer) Deploy(ctx *kingpin.ParseContext) error {
	var (
		err error
		c   clustering.Cluster
	)

	options := []clustering.Option{
		clustering.OptionDelegate(serfdom.NewLocal(cp.BitFieldMerge([]byte(nil), cp.Lurker))),
		clustering.OptionLogger(os.Stderr),
	}

	if c, err = t.global.cluster.Join(options...); err != nil {
		return err
	}

	deployment.NewDeploy(
		ux.NewTermui(t.global.cleanup, t.global.ctx),
		deploy,
		deployment.DeployerOptionChecker(status{}),
	).Deploy(c)

	// complete.
	t.shutdown()

	return err
}

type status struct{}

func (status) Check(peer *memberlist.Node) error {
	rpcClient, err := rpc.Dial("tcp", net.JoinHostPort(peer.Addr.String(), "2000"))
	if err != nil {
		log.Println("failed to connect to", peer.Name, err)
		return err
	}
	defer rpcClient.Close()
	deployClient := adapters.DeploymentClient{Client: rpcClient}
	return deployClient.Status()
}

func deploy(peer *memberlist.Node) error {
	rpcClient, err := rpc.Dial("tcp", net.JoinHostPort(peer.Addr.String(), "2000"))
	if err != nil {
		return err
	}

	return adapters.DeploymentClient{Client: rpcClient}.Deploy()
}
