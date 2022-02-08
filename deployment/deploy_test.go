package deployment_test

import (
	"errors"
	"sync/atomic"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	"github.com/james-lawrence/bw/deployment"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	completedDeploy = agent.Deploy{Stage: agent.Deploy_Completed}
	failedDeploy    = agent.Deploy{Stage: agent.Deploy_Failed}
)

var _ = Describe("Deploy", func() {
	It("should run the deploys", func() {
		var (
			deployCount int64
		)

		p := agent.NewPeer("node4")
		l := cluster.NewLocal(p)
		c := cluster.New(
			l,
			clustering.NewMock(
				func() *memberlist.Node { n := agent.PeerToNode(p); return &n }(),
				clusteringtestutil.NewNodeFromAddress("node1", "127.0.0.1"),
				clusteringtestutil.NewNodeFromAddress("node2", "127.0.0.2"),
				clusteringtestutil.NewNodeFromAddress("node3", "127.0.0.3"),
			),
		)

		deploy := deployment.NewDeploy(
			p,
			agentutil.LogDispatcher{},
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(p *agent.Peer) (ignored *agent.Deploy, err error) {
				atomic.AddInt64(&deployCount, 1)
				return ignored, nil
			})),
		)

		failures, success := deploy.Deploy(c)
		Expect(failures).To(Equal(int64(0)))
		Expect(success).To(BeTrue())
		Expect(deployCount).To(Equal(int64(len(c.Peers()))))
	})

	It("should stop when a deploy fails", func() {
		var (
			deployCount    int64
			failedDeployID int64 = 2
		)

		p := agent.NewPeer("node4")
		l := cluster.NewLocal(p)
		c := cluster.New(
			l,
			clustering.NewMock(
				func() *memberlist.Node { n := agent.PeerToNode(p); return &n }(),
				clusteringtestutil.NewNodeFromAddress("node1", "127.0.0.1"),
				clusteringtestutil.NewNodeFromAddress("node2", "127.0.0.2"),
				clusteringtestutil.NewNodeFromAddress("node3", "127.0.0.3"),
			),
		)

		deploy := deployment.NewDeploy(
			p,
			agentutil.LogDispatcher{},
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(p *agent.Peer) (ignored *agent.Deploy, err error) {
				if atomic.CompareAndSwapInt64(&deployCount, failedDeployID-1, failedDeployID) {
					return ignored, errors.New("deploy failed")
				}

				atomic.AddInt64(&deployCount, 1)

				return ignored, nil
			})),
			deployment.DeployOptionChecker(deployment.OperationFunc(func(p *agent.Peer) (ignored *agent.Deploy, err error) {
				switch atomic.LoadInt64(&deployCount) {
				case failedDeployID:
					return &failedDeploy, nil
				default:
					return &completedDeploy, nil
				}
			})),
		)

		failures, success := deploy.Deploy(c)
		Expect(failures).To(Equal(int64(1)))
		Expect(success).To(BeFalse())
		Expect(deployCount).To(Equal(int64(2)))
	})
})
