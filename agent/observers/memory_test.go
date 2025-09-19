package observers

import (
	"sync"
	"testing"

	"github.com/james-lawrence/bw/agent"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Directory", func() {
	It("should watch a directory for sockets", func() {
		dirconns, err := NewMemory()
		Expect(err).ToNot(HaveOccurred())

		l, srv, err := dirconns.Connect(make(chan *agent.Message))
		Expect(err).To(Succeed())
		defer l.Close()

		Eventually(func() int { return len(dirconns.observers) }).Should(Equal(1))
		srv.GracefulStop()
		Eventually(func() int { return len(dirconns.observers) }).Should(Equal(0))
	})
})

func BenchmarkMemoryDispatch(b *testing.B) {
	observers := make(map[string]Conn, 100)
	for i := 0; i < 100; i++ {
		observers[string(rune('a'+i/26))+string(rune('a'+i%26))] = Conn{conn: nil}
	}
	m := &sync.RWMutex{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RLock()
		for id, conn := range observers {
			_ = id
			_ = conn
		}
		m.RUnlock()
	}
}
