package clustering

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Cached", func() {
	var (
		cached *Cached
		callCount int
		mu sync.Mutex
	)

	ginkgo.BeforeEach(func() {
		callCount = 0
		cached = NewCached(func(ctx context.Context) Rendezvous {
			mu.Lock()
			callCount++
			mu.Unlock()
			return &mockRendezvous{
				nodes: []*memberlist.Node{
					{Name: "node1"},
					{Name: "node2"},
				},
			}
		})
	})

	ginkgo.It("should cache results within TTL", func() {
		result1 := cached.Get([]byte("key"))
		result2 := cached.Get([]byte("key"))

		Expect(result1).To(Equal(result2))
		Expect(callCount).To(Equal(1))
	})

	ginkgo.It("should refresh cache after TTL expires", func() {
		cached.ttl = 10 * time.Millisecond

		cached.Get([]byte("key"))
		time.Sleep(20 * time.Millisecond)
		cached.Get([]byte("key"))

		Expect(callCount).To(Equal(2))
	})

	ginkgo.It("should handle concurrent access correctly", func() {
		const numGoroutines = 100
		const numOperations = 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					cached.Get([]byte("key"))
					cached.Members()
					cached.GetN(1, []byte("key"))
				}
			}()
		}

		wg.Wait()
		Expect(callCount).To(Equal(1))
	})
})

type mockRendezvous struct {
	nodes []*memberlist.Node
}

func (m *mockRendezvous) Members() []*memberlist.Node {
	return m.nodes
}

func (m *mockRendezvous) Get(key []byte) *memberlist.Node {
	if len(m.nodes) == 0 {
		return nil
	}
	return m.nodes[0]
}

func (m *mockRendezvous) GetN(n int, key []byte) []*memberlist.Node {
	if n > len(m.nodes) {
		return m.nodes
	}
	return m.nodes[:n]
}

func createTestCacheFiller() cachefiller {
	mock := &mockRendezvous{
		nodes: []*memberlist.Node{
			{Name: "node1"},
			{Name: "node2"},
			{Name: "node3"},
		},
	}
	return func(ctx context.Context) Rendezvous {
		time.Sleep(10 * time.Microsecond)
		return mock
	}
}

func BenchmarkCachedGet(b *testing.B) {
	cached := NewCached(createTestCacheFiller())
	key := []byte("test-key")

	cached.Get(key)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cached.Get(key)
		}
	})
}

func BenchmarkCachedMembers(b *testing.B) {
	cached := NewCached(createTestCacheFiller())

	cached.Members()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cached.Members()
		}
	})
}

func BenchmarkCachedGetN(b *testing.B) {
	cached := NewCached(createTestCacheFiller())
	key := []byte("test-key")

	cached.GetN(2, key)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cached.GetN(2, key)
		}
	})
}