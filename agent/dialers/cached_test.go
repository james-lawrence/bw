package dialers_test

import (
	"context"
	"sync"
	"sync/atomic"

	. "github.com/james-lawrence/bw/agent/dialers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type countingDialer struct {
	dials *int64
}

func (t countingDialer) DialContext(ctx context.Context, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	atomic.AddInt64(t.dials, 1)
	return grpc.NewClient("passthrough:///localhost:0", grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (t countingDialer) Defaults(combined ...grpc.DialOption) Defaulted {
	return combined
}

var _ = Describe("Cached", func() {
	It("should only dial once when called concurrently", func() {
		for round := 0; round < 100; round++ {
			d := countingDialer{dials: new(int64)}
			cached := NewCached(d)

			var wg sync.WaitGroup
			start := make(chan struct{})
			for i := 0; i < 16; i++ {
				wg.Add(1)
				go func() {
					defer GinkgoRecover()
					defer wg.Done()
					<-start
					_, err := cached.DialContext(context.Background())
					Expect(err).To(Succeed())
				}()
			}
			close(start)
			wg.Wait()

			Expect(cached.Close()).To(Succeed())
			Expect(atomic.LoadInt64(d.dials)).To(Equal(int64(1)), "round %d", round)
		}
	})
})
