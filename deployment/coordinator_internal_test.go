package deployment

import (
	"os"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/storage"

	. "github.com/onsi/ginkgo/v2"

	g "github.com/onsi/gomega"
)

var _ = Describe("Coordinator", func() {
	var workdir string
	BeforeEach(func() {
		workdir = testingx.TempDir()
	})

	It("should autocorrect dead deploys", func() {
		id := bw.MustGenerateID()
		deploydir := filepath.Join(workdir, "deploys", id.String())

		g.Expect(os.MkdirAll(deploydir, 0755)).To(g.Succeed())

		p := agent.NewPeer("node1")
		c := New(
			p,
			NewDirective(),
			CoordinatorOptionRoot(workdir),
			CoordinatorOptionStorage(storage.NoopRegistry{}),
		)

		deploys, err := c.Deployments()
		g.Expect(err).To(g.Succeed())
		g.Expect(deploys).To(g.HaveLen(0))
		a := agent.Archive{
			DeploymentID: id,
			Peer:         p,
		}
		dopts := agent.DeployOptions{
			Concurrency:    1,
			IgnoreFailures: false,
			Timeout:        int64(time.Minute),
		}

		g.Expect(writeDeployMetadata(deploydir, &agent.Deploy{
			Initiator: "test user",
			Archive:   &a,
			Options:   &dopts,
			Stage:     agent.Deploy_Deploying,
		})).To(g.Succeed())

		deploys, err = c.Deployments()
		g.Expect(err).To(g.Succeed())
		g.Expect(deploys).To(g.HaveLen(1))
		g.Expect(deploys[0].Stage).To(g.Equal(agent.Deploy_Failed))
	})

	DescribeTable("Reset should properly reset deploys directory", func(s agent.Deploy_Stage, result int) {
		p := agent.NewPeer("node1")
		c := New(
			p,
			NewDirective(),
			CoordinatorOptionRoot(workdir),
			CoordinatorOptionStorage(storage.NoopRegistry{}),
		)

		deploys, err := c.Deployments()
		g.Expect(err).To(g.Succeed())
		g.Expect(deploys).To(g.HaveLen(0))

		dopts := agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    false,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}
		a := agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		deploydir := filepath.Join(workdir, "deploys", bw.RandomID(a.DeploymentID).String())
		g.Expect(os.MkdirAll(deploydir, 0755)).To(g.Succeed())
		g.Expect(writeDeployMetadata(deploydir, &agent.Deploy{
			Initiator: "test user",
			Archive:   &a,
			Options:   &dopts,
			Stage:     s,
		})).To(g.Succeed())
		deploys, err = c.Deployments()
		g.Expect(err).To(g.Succeed())
		g.Expect(deploys).To(g.HaveLen(1))

		g.Expect(c.Reset()).To(g.Succeed())

		deploys, err = c.Deployments()
		g.Expect(err).To(g.Succeed())
		g.Expect(deploys).To(g.HaveLen(result))
	},
		Entry("failed deploy should be removed", agent.Deploy_Failed, 0),
		Entry("currently deploying should be removed", agent.Deploy_Deploying, 0),
		Entry("completed deploy should remain", agent.Deploy_Completed, 1),
	)
})
