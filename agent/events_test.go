package agent_test

import (
	. "github.com/james-lawrence/bw/agent"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Events", func() {
	It("should be able to dispatch messages", func() {
		bus := NewEventBus()
		obs := &countingEventObserver{}
		obsr := bus.Register(obs)
		defer bus.Remove(obsr)
		bus.Dispatch(Message{})
		Eventually(func() []Message { return obs.seen }).Should(HaveLen(1))
	})
})
