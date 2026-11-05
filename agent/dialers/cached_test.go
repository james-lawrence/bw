package dialers_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/james-lawrence/bw/agent/dialers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCachedShouldOnlyDialOnceWhenCalledConcurrently(t *testing.T) {
	for round := 0; round < 100; round++ {
		d := countingDialer{dials: new(int64)}
		cached := NewCached(d)

		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				_, err := cached.DialContext(context.Background())
				assert.NoError(t, err)
			}()
		}
		close(start)
		wg.Wait()

		require.NoError(t, cached.Close())
		require.Equal(t, int64(1), atomic.LoadInt64(d.dials), "round %d", round)
	}
}
