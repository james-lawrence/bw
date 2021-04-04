package bootstrap_test

import (
	"context"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/agentutil"
	. "github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/internal/x/testingx"
	"google.golang.org/grpc"
)

var _ = Describe("Cluster", func() {
	var (
		peer1    = agent.NewPeer("node1")
		archive1 = agent.Archive{
			Peer:         peer1,
			Ts:           time.Now().Unix(),
			DeploymentID: bw.MustGenerateID(),
		}
		dopts1 = agent.DeployOptions{
			Timeout:           int64(time.Hour),
			SilenceDeployLogs: true,
		}
	)

	It("should succeed when no errors occur", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		p := agent.NewPeer("local")

		d, srv := testingx.NewGRPCServer2(func(srv *grpc.Server) {
			(&agenttestutil.FakeAgent{
				StatusResponse: agent.StatusResponse{
					Deployments: []*agent.Deploy{
						{
							Stage:   agent.Deploy_Completed,
							Archive: &archive1,
							Options: &dopts1,
						},
					},
				},
			}).Bind(srv)
		})
		defer testingx.GRPCCleanup(nil, srv)

		mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
		Expect(Run(context.Background(), SocketQuorum(c), NewCluster(mc, d))).To(Succeed())
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithInsecure())
		Expect(err).To(Succeed())
	})

	It("should return no deployments error when no deployments exist", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		p := agent.NewPeer("local")

		d, srv := testingx.NewGRPCServer2(func(srv *grpc.Server) {
			(&agenttestutil.FakeAgent{
				StatusResponse: agent.StatusResponse{
					Deployments: []*agent.Deploy{},
				},
			}).Bind(srv)
		})
		defer testingx.GRPCCleanup(nil, srv)

		mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
		Expect(Run(context.Background(), SocketQuorum(c), NewCluster(mc, d))).To(Succeed())
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithInsecure())
		Expect(err).To(Equal(agentutil.ErrNoDeployments))
	})

	It("should error out when an error occurrs", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		p := agent.NewPeer("local")
		d, srv := testingx.NewGRPCServer2(func(srv *grpc.Server) {
			(&agenttestutil.FakeAgent{
				ErrResult: errors.New("boom"),
			}).Bind(srv)
		})
		defer testingx.GRPCCleanup(nil, srv)

		mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
		Expect(Run(context.Background(), SocketQuorum(c), NewCluster(mc, d))).To(Succeed())
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithInsecure())
		Expect(err.Error()).To(Equal("failed to retrieve latest archive from bootstrap service: rpc error: code = Internal desc = cluster: failed to determine latest archive to bootstrap: rpc error: code = Unknown desc = boom"))
	})
})
