package deployment_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/storage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Coordinator", func() {
	var workdir string
	BeforeEach(func() {
		var err error
		workdir, err = ioutil.TempDir(".", "deployment")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(workdir)).ToNot(HaveOccurred())
	})

	It("deployments should return the deploy options", func() {
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
			deployment.CoordinatorOptionQuiet(),
		)
		deploys, err := c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(0))
		a := agent.Archive{
			Initiator:    "test user",
			DeploymentID: bw.MustGenerateID(),
			Peer:         &p,
		}
		dopts := agent.DeployOptions{
			Concurrency:    1,
			IgnoreFailures: true,
			Timeout:        int64(time.Minute),
		}

		_, err = c.Deploy(dopts, a)
		Expect(err).ToNot(HaveOccurred())

		deploys, err = c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(1))
		Expect(deploys[0].Options).ToNot(BeNil())
		opts := deploys[0].Options
		Expect(opts.Timeout).To(Equal(int64(time.Minute)))
		Expect(opts.IgnoreFailures).To(Equal(true))
	})

	It("should prevent deploys if one is already running", func() {
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
			deployment.CoordinatorOptionQuiet(),
		)
		deploys, err := c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(0))
		a := agent.Archive{
			Initiator:    "test user",
			DeploymentID: bw.MustGenerateID(),
			Peer:         &p,
		}
		dopts := agent.DeployOptions{
			Concurrency:    1,
			IgnoreFailures: false,
			Timeout:        int64(time.Minute),
		}

		_, err = c.Deploy(dopts, a)
		Expect(err).ToNot(HaveOccurred())

		deploys, err = c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(1))

		a2 := agent.Archive{
			Initiator:    "test user 2",
			DeploymentID: bw.MustGenerateID(),
			Peer:         &p,
		}
		_, err = c.Deploy(dopts, a2)
		Expect(err).To(MatchError(fmt.Sprintf("%s is already deploying: %s - Deploying", a.Initiator, bw.RandomID(a.DeploymentID).String())))

		Eventually(func() []agent.Deploy {
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
			deployment.CoordinatorOptionQuiet(),
		)
		deploys, err := c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(0))
		a := agent.Archive{
			Initiator:    "test user",
			DeploymentID: bw.MustGenerateID(),
			Peer:         &p,
		}
		dopts := agent.DeployOptions{
			Concurrency:    1,
			IgnoreFailures: false,
			Timeout:        int64(time.Minute),
		}

		_, err = c.Deploy(dopts, a)
		Expect(err).ToNot(HaveOccurred())

		c.Cancel()

		a2 := agent.Archive{
			Initiator:    "test user 2",
			DeploymentID: bw.MustGenerateID(),
			Peer:         &p,
		}
		_, err = c.Deploy(dopts, a2)
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
			deployment.CoordinatorOptionQuiet(),
		)
		deploys, err := c.Deployments()
		Expect(err).ToNot(HaveOccurred())
		Expect(deploys).To(HaveLen(0))
		a := agent.Archive{
			Initiator:    "test user",
			DeploymentID: bw.MustGenerateID(),
			Peer:         &p,
		}
		dopts := agent.DeployOptions{
			Concurrency:    1,
			IgnoreFailures: false,
			Timeout:        int64(time.Minute),
		}

		_, err = c.Deploy(dopts, a)
		Expect(err).ToNot(HaveOccurred())

		c.Cancel()
		c.Cancel()
	})
})
