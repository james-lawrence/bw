package agent_test

import (
	"context"
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

func (t *countingEventObserver) Receive(ctx context.Context, m ...agent.Message) error {
	t.seen = append(t.seen, m...)
	return nil
}

type blockingObserver struct {
	C chan struct{}
}

func (t blockingObserver) Receive(ctx context.Context, m ...agent.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
	}
	return nil
}
