package proxy

import (
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/deployment"
)

type clusterx interface {
	Local() agent.Peer
	Peers() []agent.Peer
	Quorum() []agent.Peer
	Connect() agent.ConnectInfo
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
func (t Proxy) Deploy(creds grpc.DialOption, d agent.Dispatcher, dopts agent.DeployOptions, archive agent.Archive, peers ...agent.Peer) (err error) {
	var (
		filter deployment.Filter
	)

	doptions := []grpc.DialOption{
		creds,
	}

	filter = deployment.AlwaysMatch
	if len(peers) > 0 {
		filter = deployment.Peers(peers...)
	}

	options := []deployment.Option{
		deployment.DeployOptionChecker(deployment.OperationFunc(check(doptions...))),
		deployment.DeployOptionDeployer(deployment.OperationFunc(deploy(dopts, archive, doptions...))),
		deployment.DeployOptionFilter(filter),
		deployment.DeployOptionPartitioner(bw.ConstantPartitioner(dopts.Concurrency)),
		deployment.DeployOptionIgnoreFailures(dopts.IgnoreFailures),
	}

	cmd := agent.DeployCommand{
		Command: agent.DeployCommand_Begin,
		Options: &dopts,
		Archive: &archive,
	}

	if err = d.Dispatch(agentutil.DeployCommand(t.c.Local(), cmd)); err != nil {
		return err
	}

	deployment.RunDeploy(t.c.Local(), t.c, d, options...)

	if err = d.Dispatch(agentutil.DeployCommand(t.c.Local(), agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &archive})); err != nil {
		return err
	}

	return nil
}
