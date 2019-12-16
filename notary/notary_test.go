package notary_test

import (
	"github.com/golang/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/james-lawrence/bw/notary"

	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/internal/x/testingx"
)

var _ = Describe("Notary", func() {
	Describe("Grant", func() {
		It("should fail without credentials", func() {
			notary := New("", nil, NewStorage(testingx.TempDir()))
			conn, server := testingx.NewGRPCServer(func(s *grpc.Server) {
				notary.Bind(s)
			})
			defer testingx.GRPCCleanup(conn, server)

			c := NewClient(NewStaticDialer(conn))
			_, err := c.Grant(Grant{})
			Expect(err).To(MatchError("rpc error: code = PermissionDenied desc = invalid credentials"))
		})

		It("should succeed with credentials", func() {
			c, cleanup := QuickService()
			defer cleanup()

			// generate a new key to add.
			_, npubkey := QuickKey()
			ngrant, err := c.Grant(Grant{Authorization: npubkey, Permission: PermAll()})
			Expect(err).To(Succeed())
			Expect(ngrant.Authorization).To(Equal(npubkey))
		})
	})

	Describe("Revoke", func() {
		It("should fail without credentials", func() {
			notary := New("", nil, NewStorage(testingx.TempDir()))
			conn, server := testingx.NewGRPCServer(func(s *grpc.Server) {
				notary.Bind(s)
			})
			defer testingx.GRPCCleanup(conn, server)

			c := NewClient(NewStaticDialer(conn))
			_, err := c.Revoke("")
			Expect(err).To(MatchError("rpc error: code = PermissionDenied desc = invalid credentials"))
		})

		It("should succeed with credentials", func() {
			c, cleanup := QuickService()
			defer cleanup()

			// generate a new key to add.
			_, npubkey := QuickKey()
			ngrant, err := c.Grant(Grant{Authorization: npubkey, Permission: PermAll()})
			Expect(err).To(Succeed())
			Expect(ngrant.Authorization).To(Equal(npubkey))
			nrevoked, err := c.Revoke(ngrant.Fingerprint)
			Expect(err).To(Succeed())
			Expect(proto.Equal(&nrevoked, &ngrant)).To(BeTrue())
		})
	})
})
