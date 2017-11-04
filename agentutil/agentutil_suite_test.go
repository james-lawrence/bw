package agentutil_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAgentutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agentutil Suite")
}
