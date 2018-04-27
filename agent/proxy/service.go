package proxy

import (
	"context"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/deployment"
)

type clusterx interface {
	Local() agent.Peer
	Peers() []agent.Peer
	Quorum() []agent.Peer
	Connect() agent.ConnectResponse
}

// NewProxy ...
func NewProxy(c clusterx) Proxy {
	return Proxy{
		c: c,
	}
}

// Proxy - implements the deployer.
type Proxy struct {
	c clusterx
}

// Deploy the given archive to the specified peers.
func (t Proxy) Deploy(dialer agent.Dialer, d agent.Dispatcher, dopts agent.DeployOptions, archive agent.Archive, peers ...agent.Peer) (err error) {
	var (
		filter deployment.Filter
	)

	cmd := agent.DeployCommand{
		Command: agent.DeployCommand_Begin,
		Options: &dopts,
		Archive: &archive,
	}

	if err = d.Dispatch(context.Background(), agentutil.DeployCommand(t.c.Local(), cmd)); err != nil {
		return err
	}

	filter = deployment.AlwaysMatch
	if len(peers) > 0 {
		filter = deployment.Peers(peers...)
	}

	options := []deployment.Option{
		deployment.DeployOptionChecker(deployment.OperationFunc(check(dialer))),
		deployment.DeployOptionDeployer(deployment.OperationFunc(deploy(dopts, archive, dialer))),
		deployment.DeployOptionFilter(filter),
		deployment.DeployOptionPartitioner(bw.ConstantPartitioner(dopts.Concurrency)),
		deployment.DeployOptionIgnoreFailures(dopts.IgnoreFailures),
	}

	dresult := agent.DeployCommand_Failed
	if _, success := deployment.RunDeploy(t.c.Local(), t.c, d, options...); success {
		dresult = agent.DeployCommand_Done
	}

	if err = d.Dispatch(context.Background(), agentutil.DeployCommand(t.c.Local(), agent.DeployCommand{Command: dresult, Archive: &archive})); err != nil {
		return err
	}

	return nil
}
