package bootstrap_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agenttestutil"
	. "github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/x/testingx"
	"github.com/james-lawrence/bw/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Bootstrap", func() {
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
		missing = status.Error(codes.NotFound, "missing deployment")
	)

	It("should succeed when no errors occur", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current})).To(Succeed())
		Expect(Bootstrap(context.Background(), c, dc, nil)).ToNot(HaveOccurred())
	})

	It("should fail when it fails to download the archive", func() {
		reg := storage.NoopRegistry{Err: errors.New("download failed")}
		p := agent.NewPeer("local")

		c := agent.Config{
			Root: testingx.TempDir(),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current})).To(Succeed())
		Expect(errors.Cause(Bootstrap(context.Background(), c, dc, nil))).To(MatchError("download failed"))
	})

	It("should fail when the deployment fails", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: errors.New("deployment failed")},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current})).To(Succeed())
		Expect(errors.Cause(Bootstrap(context.Background(), c, dc, nil))).To(MatchError("deployment failed"))
	})

	It("should succeed when it finishes bootstrapping from quorum", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current})).To(Succeed())
		Expect(Bootstrap(context.Background(), c, dc, nil)).ToNot(HaveOccurred())
	})

	It("should bootstrap from fallback bootstrap services when quorum has no deployments", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Run(context.Background(), SocketAuto(c), &agenttestutil.Mock{Current: current})).To(Succeed())
		results := make(chan *deployment.DeployResult, 1)
		Expect(Bootstrap(context.Background(), c, dc, results)).To(MatchError("failed to determine latest deployment from quorum, retrying: no deployments found"))
		result := <-results
		Expect([]byte(result.ID)).To(Equal(current.Archive.DeploymentID))
	})

	It("should stop attempting to bootstrap if all services return no deployments found", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}

		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Run(context.Background(), SocketAuto(c), &agenttestutil.Mock{Fail: missing})).To(Succeed())
		Expect(Bootstrap(context.Background(), c, dc, nil)).To(Succeed())
	})

	Context("active deploy", func() {
		It("should deploy an active deploy from quorum", func() {
			c := agent.Config{
				Root: testingx.TempDir(),
			}
			current := &agent.Deploy{
				Stage:   agent.Deploy_Completed,
				Archive: &archive1,
				Options: &dopts1,
			}
			reg := storage.NoopRegistry{}
			p := agent.NewPeer("local")

			dc := deployment.New(
				p,
				noopDeployer{err: nil},
				deployment.CoordinatorOptionStorage(reg),
				deployment.CoordinatorOptionRoot(testingx.TempDir()),
			)
			Expect(Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Current: current})).To(Succeed())
			Expect(Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current, Info: agent.ArchiveResponse_ActiveDeploy})).To(Succeed())
			Expect(Bootstrap(context.Background(), c, dc, nil).Error()).To(Equal("active deploy matches the local deployment, waiting for deployment to complete: deployment in progress"))
		})
	})
})
