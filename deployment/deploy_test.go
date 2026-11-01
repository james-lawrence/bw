package deployment_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

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

// recorder is a dispatcher that records every message.
type recorder struct {
	m        sync.Mutex
	messages []*agent.Message
}

func (t *recorder) Dispatch(_ context.Context, messages ...*agent.Message) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.messages = append(t.messages, messages...)
	return nil
}

func (t *recorder) ofType(typ agent.Message_Type) (found []*agent.Message) {
	t.m.Lock()
	defer t.m.Unlock()
	for _, m := range t.messages {
		if m.Type == typ {
			found = append(found, m)
		}
	}
	return found
}

func (t *recorder) logs() (found []string) {
	for _, m := range t.ofType(agent.Message_LogEvent) {
		found = append(found, m.GetLog().Log)
	}
	return found
}

func testCluster(p *agent.Peer) cluster.Cluster {
	return cluster.New(
		p,
		clustering.NewMock(
			agent.PeerToNode(p),
			clusteringtestutil.NewNodeFromAddress("node1", "127.0.0.1"),
			clusteringtestutil.NewNodeFromAddress("node2", "127.0.0.2"),
			clusteringtestutil.NewNodeFromAddress("node3", "127.0.0.3"),
		),
	)
}

var _ = Describe("Deploy", func() {
	It("should run the deploys", func() {
		var (
			deployCount int64
		)

		p := agent.NewPeer("node4")
		c := cluster.New(
			p,
			clustering.NewMock(
				agent.PeerToNode(p),
				clusteringtestutil.NewNodeFromAddress("node1", "127.0.0.1"),
				clusteringtestutil.NewNodeFromAddress("node2", "127.0.0.2"),
				clusteringtestutil.NewNodeFromAddress("node3", "127.0.0.3"),
			),
		)

		deploy := deployment.NewDeploy(
			p,
			agentutil.DiscardDispatcher{},
			deployment.DeployOptionTimeout(100*time.Millisecond),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				atomic.AddInt64(&deployCount, 1)
				return ignored, nil
			})),
		)

		failures, success := deploy.Deploy(c)
		Expect(failures).To(Equal(int64(0)))
		Expect(success).To(BeTrue())
		Expect(deployCount).To(Equal(int64(len(c.Peers()))))
	})

	It("should be durable against healthcheck failures", func() {
		var (
			deploycount      int64
			healthcheckcount int64
		)

		p := agent.NewPeer("node4")
		c := cluster.New(
			p,
			clustering.NewMock(
				agent.PeerToNode(p),
				clusteringtestutil.NewNodeFromAddress("node1", "127.0.0.1"),
				clusteringtestutil.NewNodeFromAddress("node2", "127.0.0.2"),
				clusteringtestutil.NewNodeFromAddress("node3", "127.0.0.3"),
			),
		)

		deploy := deployment.NewDeploy(
			p,
			agentutil.DiscardDispatcher{},
			deployment.DeployOptionTimeout(100*time.Millisecond),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				atomic.AddInt64(&deploycount, 1)
				return ignored, nil
			})),
			deployment.DeployOptionChecker(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				switch atomic.AddInt64(&healthcheckcount, 1) {
				case 1:
					return &failedDeploy, nil
				default:
					return &completedDeploy, nil
				}
			})),
		)

		failures, success := deploy.Deploy(c)
		Expect(failures).To(Equal(int64(0)))
		Expect(success).To(BeTrue())
		Expect(deploycount).To(Equal(int64(4)))
	})

	It("should stop when a deploy fails", func() {
		var (
			deployCount    int64
			failedDeployID int64 = 2
		)

		p := agent.NewPeer("node4")
		c := cluster.New(
			p,
			clustering.NewMock(
				agent.PeerToNode(p),
				clusteringtestutil.NewNodeFromAddress("node1", "127.0.0.1"),
				clusteringtestutil.NewNodeFromAddress("node2", "127.0.0.2"),
				clusteringtestutil.NewNodeFromAddress("node3", "127.0.0.3"),
			),
		)

		deploy := deployment.NewDeploy(
			p,
			agentutil.DiscardDispatcher{},
			deployment.DeployOptionTimeout(100*time.Millisecond),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				atomic.AddInt64(&deployCount, 1)
				return ignored, nil
			})),
			deployment.DeployOptionChecker(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
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
	It("should report progress events while deploying", func() {
		p := agent.NewPeer("node4")
		c := testCluster(p)
		events := &recorder{}

		deploy := deployment.NewDeploy(
			p,
			events,
			deployment.DeployOptionTimeout(100*time.Millisecond),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				return ignored, nil
			})),
		)

		failures, success := deploy.Deploy(c)
		Expect(failures).To(Equal(int64(0)))
		Expect(success).To(BeTrue())

		found := events.ofType(agent.Message_PeersFoundEvent)
		Expect(found).To(HaveLen(1))
		Expect(found[0].GetInt()).To(Equal(int64(len(c.Peers()))))
		Expect(events.logs()).To(ContainElement("nodes are ready, deploying"))

		completed := events.ofType(agent.Message_PeersCompletedEvent)
		Expect(completed).To(HaveLen(len(c.Peers())))
		Expect(completed[len(completed)-1].GetInt()).To(Equal(int64(len(c.Peers()))))
	})

	It("should only deploy to peers matching the filter", func() {
		var (
			mu       sync.Mutex
			deployed []string
		)

		p := agent.NewPeer("node4")
		c := testCluster(p)
		target := c.Peers()[0]

		deploy := deployment.NewDeploy(
			p,
			agentutil.DiscardDispatcher{},
			deployment.DeployOptionTimeout(100*time.Millisecond),
			deployment.DeployOptionFilter(deployment.Peers(target)),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				mu.Lock()
				defer mu.Unlock()
				deployed = append(deployed, p.Name)
				return ignored, nil
			})),
		)

		failures, success := deploy.Deploy(c)
		Expect(failures).To(Equal(int64(0)))
		Expect(success).To(BeTrue())
		Expect(deployed).To(Equal([]string{target.Name}))
	})

	It("should fail when the deployer returns an error", func() {
		p := agent.NewPeer("node4")
		events := &recorder{}

		deploy := deployment.NewDeploy(
			p,
			events,
			deployment.DeployOptionTimeout(100*time.Millisecond),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				return ignored, errors.New("boom")
			})),
		)

		failures, success := deploy.Deploy(testCluster(p))
		Expect(failures).To(BeNumerically(">=", 1))
		Expect(success).To(BeFalse())
		Expect(events.ofType(agent.Message_LogEvent)).ToNot(BeEmpty())
	})

	It("should continue past failures when failures are ignored", func() {
		var (
			deployCount int64
		)

		p := agent.NewPeer("node4")
		c := testCluster(p)

		deploy := deployment.NewDeploy(
			p,
			agentutil.DiscardDispatcher{},
			deployment.DeployOptionTimeout(100*time.Millisecond),
			deployment.DeployOptionIgnoreFailures(true),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				if atomic.AddInt64(&deployCount, 1) == 2 {
					return ignored, errors.New("boom")
				}
				return ignored, nil
			})),
		)

		failures, success := deploy.Deploy(c)
		Expect(failures).To(Equal(int64(1)))
		Expect(success).To(BeTrue())
		Expect(deployCount).To(Equal(int64(len(c.Peers()))))
	})

	It("should emit heartbeats until the deploy completes", func() {
		p := agent.NewPeer("node4")
		events := &recorder{}

		deploy := deployment.NewDeploy(
			p,
			events,
			deployment.DeployOptionTimeout(5*time.Second),
			deployment.DeployOptionHeartbeatFrequency(10*time.Millisecond),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				time.Sleep(100 * time.Millisecond)
				return ignored, nil
			})),
		)

		failures, success := deploy.Deploy(testCluster(p))
		Expect(failures).To(Equal(int64(0)))
		Expect(success).To(BeTrue())
		Expect(len(events.ofType(agent.Message_DeployHeartbeat))).To(BeNumerically(">=", 3))

		// the heartbeat stops once the deploy is complete.
		settled := len(events.ofType(agent.Message_DeployHeartbeat))
		time.Sleep(50 * time.Millisecond)
		Expect(events.ofType(agent.Message_DeployHeartbeat)).To(HaveLen(settled))
	})

	It("should give up once the timeout and grace period elapse", func() {
		p := agent.NewPeer("node4")
		events := &recorder{}

		deploy := deployment.NewDeploy(
			p,
			events,
			deployment.DeployOptionTimeout(50*time.Millisecond),
			deployment.DeployOptionGracePeriod(50*time.Millisecond),
			deployment.DeployOptionChecker(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				return &agent.Deploy{Stage: agent.Deploy_Deploying}, nil
			})),
		)

		result := make(chan bool, 1)
		go func() {
			defer GinkgoRecover()
			_, success := deploy.Deploy(testCluster(p))
			result <- success
		}()

		Eventually(result, 5*time.Second).Should(Receive(BeFalse()))
	})
})
