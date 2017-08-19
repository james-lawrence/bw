package deployment_test

import (
	. "bitbucket.org/jatone/bearded-wookie/deployment"

	. "github.com/onsi/ginkgo"
	gomega "github.com/onsi/gomega"
)

var _ = Describe("Coordinator", func() {
	var coordinator Coordinator
	BeforeEach(func() {
		coordinator = NewDummyCoordinator()
	})

	Describe("Status", func() {
		It("returns nil", func() {
			gomega.Expect(coordinator.Status()).To(gomega.BeNil())
		})
	})
})
