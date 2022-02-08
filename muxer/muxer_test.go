package muxer_test

import (
	"context"
	"net"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"

	. "github.com/james-lawrence/bw/muxer"
)

var _ = Describe("ParseURI", func() {
	It("should extract the procol name and address", func() {
		proto, host, err := ParseURI("bw.agent://QmNMbFdN9s7R1PBr9VqZqVrsZzunwh61hD1vTR96AmohGX")
		Expect(err).To(Succeed())
		Expect(string(proto)).To(Equal("bw.agent"))
		Expect(host).To(Equal("QmNMbFdN9s7R1PBr9VqZqVrsZzunwh61hD1vTR96AmohGX"))
	})
})

var _ = Describe("Rebind", func() {
	It("should be able to rebind a protocol successfully", func() {
		c1 := int64(0)
		c2 := int64(0)
		counter := func(l net.Listener, c *int64) {
			for {
				_, err := l.Accept()
				if err != nil {
					return
				}
				atomic.AddInt64(c, 1)
			}
		}

		m := New()
		l, err := net.Listen("tcp", ":0")
		go func() {
			Listen(context.Background(), m, l)
		}()
		Expect(err).To(Succeed())
		defer l.Close()

		l1, err := m.Rebind("proto", l.Addr())
		Expect(err).To(Succeed())
		defer l1.Close()
		go counter(l1, &c1)
		l2, err := m.Rebind("proto", l.Addr())
		Expect(err).To(Succeed())
		defer l2.Close()
		go counter(l2, &c2)

		d1 := NewDialer("proto", &net.Dialer{})

		for i := 0; i < 10; i++ {
			conn, err := d1.DialContext(context.Background(), "tcp", l.Addr().String())
			Expect(err).To(Succeed())
			Expect(conn.Close()).To(Succeed())
			Expect(atomic.LoadInt64(&c1)).To(Equal(int64(0)))
			Expect(atomic.LoadInt64(&c2)).To(Equal(int64(i + 1)))
		}
	})
})

var _ = Describe("Dial and accept", func() {
	It("should be able to multiplex", func() {
		c1 := int64(0)
		c2 := int64(0)
		counter := func(l net.Listener, c *int64) {
			for {
				_, err := l.Accept()
				if err != nil {
					return
				}
				atomic.AddInt64(c, 1)
			}
		}

		m := New()
		l, err := net.Listen("tcp", ":0")
		go func() {
			Listen(context.Background(), m, l)
		}()
		Expect(err).To(Succeed())
		defer l.Close()

		l1, err := m.Bind("proto1", l.Addr())
		Expect(err).To(Succeed())
		go counter(l1, &c1)
		l2, err := m.Bind("proto2", l.Addr())
		Expect(err).To(Succeed())
		go counter(l2, &c2)

		d1 := NewDialer("proto1", &net.Dialer{})

		conn, err := d1.DialContext(context.Background(), "tcp", l.Addr().String())
		Expect(err).To(Succeed())
		Expect(conn.Close()).To(Succeed())
		Expect(atomic.LoadInt64(&c1)).To(Equal(int64(1)))
		Expect(atomic.LoadInt64(&c2)).To(Equal(int64(0)))

		d2 := NewDialer("proto2", &net.Dialer{})

		conn, err = d2.DialContext(context.Background(), "tcp", l.Addr().String())
		Expect(err).To(Succeed())
		Expect(conn.Close()).To(Succeed())
		Expect(atomic.LoadInt64(&c1)).To(Equal(int64(1)))
		Expect(atomic.LoadInt64(&c2)).To(Equal(int64(1)))
	})
})

var _ = Describe("Accepted", func() {
	DescribeTable("should encode to a fixed size (22)", func(name string, code AcceptedError) {
		protocol := Proto(name)
		encoded, err := proto.Marshal(&Accepted{
			Version:  1,
			Code:     code,
			Protocol: protocol[:],
		})
		Expect(err).To(Succeed())
		Expect(len(encoded)).To(Equal(22))
	},
		Entry("example 1 - default protocol", "", Accepted_None),
		Entry("example 2 - long name", "45b3058c-9ec4-41cc-bd4c-c74ca2abdea2", Accepted_None),
		Entry("example 3 - client error", "45b3058c-9ec4-41cc-bd4c-c74ca2abdea2", Accepted_ClientError),
		Entry("example 4 - server error", "45b3058c-9ec4-41cc-bd4c-c74ca2abdea2", Accepted_ServerError),
	)
})

var _ = Describe("Requested", func() {
	DescribeTable("should encode to a fixed size (20)", func(name string) {
		protocol := Proto(name)
		encoded, err := proto.Marshal(&Requested{
			Version:  1,
			Protocol: protocol[:],
		})
		Expect(err).To(Succeed())
		Expect(len(encoded)).To(Equal(20))
	},
		Entry("example 1 - default protocol", ""),
		Entry("example 2 - long name", "45b3058c-9ec4-41cc-bd4c-c74ca2abdea2"),
	)
})
