package agentutil_test

import (
	"io/ioutil"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAgentutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agentutil Suite")
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})
