package proxy

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/deployment"
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
func (t Proxy) Deploy(max int64, creds credentials.TransportCredentials, info agent.Archive, peers ...agent.Peer) {
	var (
		filter deployment.Filter
	)

	doptions := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	filter = deployment.AlwaysMatch
	if len(peers) > 0 {
		filter = deployment.Peers(peers...)
	}

	options := []deployment.Option{
		deployment.DeployOptionChecker(deployment.OperationFunc(check(doptions...))),
		deployment.DeployOptionDeployer(deployment.OperationFunc(deploy(info, doptions...))),
		deployment.DeployOptionFilter(filter),
		deployment.DeployOptionPartitioner(bw.ConstantPartitioner{BatchMax: int(max)}),
	}

	deployment.NewDeploy(
		t.c.Local(),
		t.d,
		options...,
	).Deploy(t.c)
}
