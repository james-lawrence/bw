package agent_test

import (
	"bitbucket.org/jatone/bearded-wookie/agent"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Suite")
}

func mustCommand(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

type countingEventObserver struct {
	seen []agent.Message
}

func (t *countingEventObserver) Receive(m ...agent.Message) error {
	t.seen = append(t.seen, m...)
	return nil
}
