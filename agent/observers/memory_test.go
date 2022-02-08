package observers

import (
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
