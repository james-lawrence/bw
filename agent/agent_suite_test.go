package agent_test

import (
	"context"
	"io"
	"log"

	"github.com/james-lawrence/bw/agent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAgent(t *testing.T) {
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Suite")
}

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
