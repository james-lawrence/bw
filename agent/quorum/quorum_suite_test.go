package quorum_test

import (
	"github.com/james-lawrence/bw/agent"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestQuorum(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Quorum Suite")
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
