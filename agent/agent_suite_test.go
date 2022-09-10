package agent_test

import (
	"io"
	"log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAgent(t *testing.T) {
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Suite")
}
