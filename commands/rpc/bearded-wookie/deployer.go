package main

import (
	"context"
	"log"
	"net"
	"net/rpc"

	cp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/ux"

	"github.com/hashicorp/memberlist"
	"github.com/alecthomas/kingpin"
)

type deployer struct {
	*cluster
	ctx    context.Context
	cancel context.CancelFunc
}

func (t *deployer) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(parent)
	parent.Command("all", "deploy to all nodes within the cluster").Default().Action(t.Deploy)
}

func (t *deployer) Deploy(ctx *kingpin.ParseContext) error {
	var (
		err error
	)
	coptions := []serfdom.ClusterOption{
		serfdom.CODelegate(serfdom.NewLocal(cp.BitFieldMerge([]byte(nil), cp.Lurker))),
	}

	if err = t.cluster.Join(ctx, coptions...); err != nil {
		return err
	}

	deployment.NewDeploy(
		// ux.Logging(),
		ux.NewTermui(t.ctx),
		deploy,
		deployment.DeployerOptionChecker(status{}),
	).Deploy(t.cluster.memberlist)

	// complete.
	t.cancel()

	return nil
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
