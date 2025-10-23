package deployment_test

import (
	"context"
	"fmt"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/storage"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Coordinator", func() {
	var workdir string
	BeforeEach(func() {
		workdir = testingx.TempDir()
	})

	It("deployments should return the deploy options", func() {
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		deploys, err := c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(0))
		a := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		dopts := &agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    true,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}

		_, err = c.Deploy(context.Background(), "test user", dopts, a)
		Expect(err).ToNot(HaveOccurred())

		deploys, err = c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(1))
		Expect(deploys[0].Options).ToNot(BeNil())
		opts := deploys[0].Options
		Expect(opts.Timeout).To(Equal(int64(time.Minute)))
		Expect(opts.IgnoreFailures).To(Equal(true))
	})

	PIt("should prevent deploys if one is already running", func() {
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		deploys, err := c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(0))
		a := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		dopts := &agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    false,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}

		_, err = c.Deploy(context.Background(), "test user", dopts, a)
		Expect(err).ToNot(HaveOccurred())

		deploys, err = c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(1))

		a2 := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		_, err = c.Deploy(context.Background(), "test user 2", dopts, a2)
		Expect(err).To(MatchError(fmt.Sprintf("test user is already deploying: %s - Deploying", bw.RandomID(a.DeploymentID).String())))

		Eventually(func() []*agent.Deploy {
			deploys, err := c.Deployments()
			Expect(err).ToNot(HaveOccurred())
			return deploys
		}).Should(HaveLen(2))
	})

	It("should be able to cancel a running deploy", func() {
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		deploys, err := c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(0))
		a := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		dopts := &agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    false,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}

		_, err = c.Deploy(context.Background(), "test user", dopts, a)
		Expect(err).ToNot(HaveOccurred())

		c.Cancel()

		a2 := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		_, err = c.Deploy(context.Background(), "test user 2", dopts, a2)
		Expect(err).ToNot(HaveOccurred())

		deploys, err = c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(2))
	})

	It("should be safe to cancel a deploy multiple times", func() {
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		deploys, err := c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(0))
		a := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		dopts := &agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    false,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}

		_, err = c.Deploy(context.Background(), "test user", dopts, a)
		Expect(err).ToNot(HaveOccurred())

		c.Cancel()
		c.Cancel()
	})
})
