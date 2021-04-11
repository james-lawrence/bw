package raftutil

import (
	"crypto/tls"
	"net"
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

		l, err := net.Listen("tcp", "127.1.1.2:9999")
		Expect(err).To(Succeed())
		sl := NewStreamTransport(l, NewTLSStreamDialer(c))

		nt := raft.NewNetworkTransport(sl, 5, 10*time.Second, os.Stderr)
		Expect(autocloseTransport(nt)).To(Succeed())

		l, err = net.Listen("tcp", "127.1.1.2:9999")
		Expect(err).To(Succeed())

		sl = NewStreamTransport(l, NewTLSStreamDialer(c))
		nt = raft.NewNetworkTransport(sl, 5, 10*time.Second, os.Stderr)
		Expect(autocloseTransport(nt)).To(Succeed())
	})
})
