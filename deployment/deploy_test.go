package deployment_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	"github.com/james-lawrence/bw/deployment"
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

func TestDeploy(t *testing.T) {
	t.Run("should run the deploys", func(t *testing.T) {
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
		require.Equal(t, int64(0), failures)
		require.True(t, success)
		require.Equal(t, int64(len(c.Peers())), deployCount)
	})

	t.Run("should be durable against healthcheck failures", func(t *testing.T) {
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
		require.Equal(t, int64(0), failures)
		require.True(t, success)
		require.Equal(t, int64(4), deploycount)
	})

	t.Run("should stop when a deploy fails", func(t *testing.T) {
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
		require.Equal(t, int64(1), failures)
		require.False(t, success)
		require.Equal(t, int64(2), deployCount)
	})
	t.Run("should report progress events while deploying", func(t *testing.T) {
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
		require.Equal(t, int64(0), failures)
		require.True(t, success)

		found := events.ofType(agent.Message_PeersFoundEvent)
		require.Len(t, found, 1)
		require.Equal(t, int64(len(c.Peers())), found[0].GetInt())
		require.Contains(t, events.logs(), "nodes are ready, deploying")

		completed := events.ofType(agent.Message_PeersCompletedEvent)
		require.Len(t, completed, len(c.Peers()))
		require.Equal(t, int64(len(c.Peers())), completed[len(completed)-1].GetInt())
	})

	t.Run("should only deploy to peers matching the filter", func(t *testing.T) {
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
		require.Equal(t, int64(0), failures)
		require.True(t, success)
		require.Equal(t, []string{target.Name}, deployed)
	})

	t.Run("should fail when the deployer returns an error", func(t *testing.T) {
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
		require.GreaterOrEqual(t, failures, int64(1))
		require.False(t, success)
		require.NotEmpty(t, events.ofType(agent.Message_LogEvent))
	})

	t.Run("should continue past failures when failures are ignored", func(t *testing.T) {
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
		require.Equal(t, int64(1), failures)
		require.True(t, success)
		require.Equal(t, int64(len(c.Peers())), deployCount)
	})

	t.Run("should emit heartbeats until the deploy completes", func(t *testing.T) {
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
		require.Equal(t, int64(0), failures)
		require.True(t, success)
		require.GreaterOrEqual(t, len(events.ofType(agent.Message_DeployHeartbeat)), 3)

		// the heartbeat stops once the deploy is complete.
		settled := len(events.ofType(agent.Message_DeployHeartbeat))
		time.Sleep(50 * time.Millisecond)
		require.Len(t, events.ofType(agent.Message_DeployHeartbeat), settled)
	})

	t.Run("should give up once the timeout and grace period elapse", func(t *testing.T) {
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
			_, success := deploy.Deploy(testCluster(p))
			result <- success
		}()

		select {
		case success := <-result:
			require.False(t, success)
		case <-time.After(5 * time.Second):
			require.FailNow(t, "timed out waiting for the deploy to give up")
		}
		require.True(t, slices.ContainsFunc(events.logs(), func(l string) bool {
			return strings.Contains(l, "timed out while waiting for nodes to complete")
		}))
	})
}
