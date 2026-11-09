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

// contextFixture is a single node cluster whose nodes never finish, only the context can end the deploy before the timeout.
func contextFixture() (*agent.Peer, cluster.Cluster, deployment.OperationFunc) {
	p := agent.NewPeer("node4")
	c := cluster.New(
		p,
		clustering.NewMock(
			agent.PeerToNode(p),
			clusteringtestutil.NewNodeFromAddress("node1", "127.0.0.1"),
		),
	)

	return p, c, deployment.OperationFunc(func(context.Context, *agent.Peer) (*agent.Deploy, error) {
		return &agent.Deploy{Stage: agent.Deploy_Deploying}, nil
	})
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

		failures, success := deploy.Deploy(t.Context(), c)
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
			deployment.DeployOptionReadiness(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				switch atomic.AddInt64(&healthcheckcount, 1) {
				case 1:
					return &failedDeploy, nil
				default:
					return &completedDeploy, nil
				}
			})),
		)

		failures, success := deploy.Deploy(t.Context(), c)
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

		failures, success := deploy.Deploy(t.Context(), c)
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

		failures, success := deploy.Deploy(t.Context(), c)
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

		failures, success := deploy.Deploy(t.Context(), c)
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

		failures, success := deploy.Deploy(t.Context(), testCluster(p))
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

		failures, success := deploy.Deploy(t.Context(), c)
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

		failures, success := deploy.Deploy(t.Context(), testCluster(p))
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
			_, success := deploy.Deploy(t.Context(), testCluster(p))
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

	t.Run("cancelling the context stops the deploy and is not a success", func(t *testing.T) {
		p, c, stuck := contextFixture()
		ctx, cancel := context.WithCancelCause(t.Context())
		rec := &recorder{}

		deploy := deployment.NewDeploy(
			p,
			rec,
			deployment.DeployOptionTimeout(time.Minute),
			deployment.DeployOptionChecker(stuck),
			deployment.DeployOptionMonitor(deployment.NewMonitor(deployment.MonitorTicklerPeriodic(10*time.Millisecond))),
		)

		time.AfterFunc(100*time.Millisecond, func() { cancel(errors.New("superseded")) })

		done := make(chan bool, 1)
		go func() {
			_, success := deploy.Deploy(ctx, c)
			done <- success
		}()

		select {
		case success := <-done:
			require.False(t, success)
		case <-time.After(5 * time.Second):
			t.Fatal("the deploy kept running after its context was cancelled")
		}

		var reason bool
		for _, l := range rec.logs() {
			reason = reason || strings.Contains(l, "superseded")
		}
		require.True(t, reason, "the cause of the stop should be recorded")
	})

	t.Run("a context cancelled before the deploy starts runs nothing", func(t *testing.T) {
		p, c, _ := contextFixture()
		ctx, cancel := context.WithCancelCause(t.Context())
		cancel(errors.New("leadership lost"))

		var visits int64
		deploy := deployment.NewDeploy(
			p,
			agentutil.DiscardDispatcher{},
			deployment.DeployOptionTimeout(time.Minute),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(context.Context, *agent.Peer) (*agent.Deploy, error) {
				atomic.AddInt64(&visits, 1)
				return &agent.Deploy{Stage: agent.Deploy_Deploying}, nil
			})),
		)

		_, success := deploy.Deploy(ctx, c)
		require.False(t, success)
		require.Zero(t, atomic.LoadInt64(&visits))
	})

	// the checker used while deploying only finds the deployment once the node has been sent it. the cluster must
	// still be deployed to, waiting for nodes to know about the deployment before sending it to them is a deadlock.
	t.Run("checks only know the deployment once sent", func(t *testing.T) {
		var (
			m    sync.Mutex
			sent = map[string]bool{}
		)

		p := agent.NewPeer("node4")
		c := testCluster(p)

		deploy := deployment.NewDeploy(
			p,
			agentutil.DiscardDispatcher{},
			deployment.DeployOptionTimeout(time.Second),
			deployment.DeployOptionHeartbeatFrequency(time.Hour),
			deployment.DeployOptionDeployer(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (ignored *agent.Deploy, err error) {
				m.Lock()
				defer m.Unlock()
				sent[p.Ip] = true
				return ignored, nil
			})),
			deployment.DeployOptionChecker(deployment.OperationFunc(func(ctx context.Context, p *agent.Peer) (*agent.Deploy, error) {
				m.Lock()
				defer m.Unlock()
				if !sent[p.Ip] {
					return nil, errors.New("deployment not found on " + p.Ip)
				}
				return &completedDeploy, nil
			})),
		)

		ctx, done := context.WithTimeout(t.Context(), 10*time.Second)
		defer done()

		failures, success := deploy.Deploy(ctx, c)
		require.NoError(t, ctx.Err(), "deploy hung waiting for the nodes to know about a deployment they were never sent")
		require.Equal(t, int64(0), failures)
		require.True(t, success)
		require.Len(t, sent, 4)
	})
}
