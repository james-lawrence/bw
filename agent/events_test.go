package agent_test

import (
	"context"
	"time"

	. "github.com/james-lawrence/bw/agent"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Events", func() {
	It("should be able to dispatch messages", func() {
		bus := NewEventBusDefault()
		obs := &countingEventObserver{}
		obsr := bus.Register(obs)
		defer bus.Remove(obsr)
		bus.Dispatch(context.Background(), Message{})
		Eventually(func() []Message { return obs.seen }).Should(HaveLen(1))
	})

	It("should eventually timeout with a bad observer", func() {
		bus := NewEventBus(make(chan []Message))
		obs := blockingObserver{}
		obsr := bus.Register(obs)
		defer bus.Remove(obsr)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		Expect(bus.Dispatch(ctx, Message{}, Message{})).ToNot(HaveOccurred())
		Expect(bus.Dispatch(ctx, Message{}, Message{}, Message{})).To(HaveOccurred())
	})
})
