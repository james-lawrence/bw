package deployment_test

import (
	. "bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Coordinator", func() {
	var coordinator Coordinator
	BeforeEach(func() {
		coordinator = NewDummyCoordinator(agent.Peer{})
	})
})
