package raftutil

import (
	"crypto/tls"
	"os"
	"time"

	"github.com/hashicorp/raft"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TLSTransport", func() {
	It("should gracefully close and reopen a net listenr", func() {
		c := &tls.Config{
			InsecureSkipVerify: true,
		}

		sl, err := NewTLSTCP("127.1.1.2:9999", c)
		Expect(err).To(Succeed())

		// check that we do fail without closing.
		_, err = NewTLSTCP("127.1.1.2:9999", c)
		Expect(err).ToNot(Succeed())

		nt := raft.NewNetworkTransport(sl, 5, 10*time.Second, os.Stderr)
		Expect(autocloseTransport(nt)).To(Succeed())

		sl, err = NewTLSTCP("127.1.1.2:9999", c)
		Expect(err).To(Succeed())
		nt = raft.NewNetworkTransport(sl, 5, 10*time.Second, os.Stderr)
		Expect(autocloseTransport(nt)).To(Succeed())
	})
})
