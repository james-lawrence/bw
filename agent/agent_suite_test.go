package agent_test

import (
	"io/ioutil"
	"log"

	"github.com/james-lawrence/bw/agent"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Suite")
}

var _ = BeforeSuite(func() {
	log.SetOutput(ioutil.Discard)
})

type countingEventObserver struct {
	seen []agent.Message
}

func (t *countingEventObserver) Receive(m ...agent.Message) error {
	t.seen = append(t.seen, m...)
	return nil
}
