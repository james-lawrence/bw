package deployment

import (
	"os"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/testingx"
	"github.com/james-lawrence/bw/storage"

	. "github.com/onsi/ginkgo"
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
			Initiator:    "test user",
			DeploymentID: id,
			Peer:         &p,
		}
		dopts := agent.DeployOptions{
			Concurrency:    1,
			IgnoreFailures: false,
			Timeout:        int64(time.Minute),
		}

		g.Expect(writeDeployMetadata(deploydir, agent.Deploy{Archive: &a, Options: &dopts, Stage: agent.Deploy_Deploying})).To(g.Succeed())

		// takes two requests for correction to take effect,
		// first request corrects the invalid deploy, second reads from it.
		deploys, err = c.Deployments()
		g.Expect(err).To(g.Succeed())
		g.Expect(deploys).To(g.HaveLen(1))
		g.Expect(deploys[0].Stage).To(g.Equal(agent.Deploy_Deploying))

		deploys, err = c.Deployments()
		g.Expect(err).To(g.Succeed())
		g.Expect(deploys).To(g.HaveLen(1))
		g.Expect(deploys[0].Stage).To(g.Equal(agent.Deploy_Failed))
	})
})
