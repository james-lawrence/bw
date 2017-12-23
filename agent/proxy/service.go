package proxy

import (
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
)

type clusterx interface {
	Local() agent.Peer
	Peers() []agent.Peer
	Quorum() []agent.Peer
	Connect() agent.ConnectInfo
}

// NewProxy ...
func NewProxy(c clusterx, d dispatcher) Proxy {
	return Proxy{
		c: c,
		d: d,
	}
}

// Proxy - implements the deployer.
type Proxy struct {
	c clusterx
	d dispatcher
}

// Deploy ...
func (t Proxy) Deploy(max int64, creds grpc.DialOption, info agent.Archive, peers ...agent.Peer) {
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
		deployment.DeployOptionDeployer(deployment.OperationFunc(deploy(info, doptions...))),
		deployment.DeployOptionFilter(filter),
		deployment.DeployOptionPartitioner(bw.ConstantPartitioner(int(max))),
	}

	deployment.NewDeploy(
		t.c.Local(),
		t.d,
		options...,
	).Deploy(t.c)
}
