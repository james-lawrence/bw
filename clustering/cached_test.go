package clustering

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCached(t *testing.T) {
	counting := func() (*int, cachefiller) {
		fetches := new(int)
		return fetches, func(context.Context) Rendezvous {
			*fetches++
			return NewStatic()
		}
	}

	t.Run("fetches once within the ttl", func(t *testing.T) {
		fetches, fetch := counting()
		c := NewCached(fetch)

		for i := 0; i < 5; i++ {
			c.Members()
			c.Get([]byte("key"))
			c.GetN(1, []byte("key"))
		}

		require.Equal(t, 1, *fetches)
	})

	t.Run("fetches again once the ttl has passed", func(t *testing.T) {
		fetches, fetch := counting()
		c := NewCached(fetch)
		c.ttl = 10 * time.Millisecond

		c.Members()
		time.Sleep(50 * time.Millisecond)
		c.Members()

		require.Equal(t, 2, *fetches)
	})
}
