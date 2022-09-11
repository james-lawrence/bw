package notary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"

	. "github.com/james-lawrence/bw/notary"

	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/internal/testingx"
)

var _ = Describe("Notary", func() {
	Describe("Grant", func() {
		It("should fail without credentials", func() {
			notary := New("", nil, NewDirectory(testingx.TempDir()))
			conn, server := testingx.NewGRPCServer(func(s *grpc.Server) {
				notary.Bind(s)
			})
			defer testingx.GRPCCleanup(conn, server)

			c := NewClient(NewStaticDialer(conn))
			_, err := c.Grant(&Grant{})
			Expect(err).To(MatchError("rpc error: code = PermissionDenied desc = invalid credentials"))
		})

		It("should succeed with credentials", func() {
			c, cleanup := QuickService()
			defer cleanup()

			// generate a new key to add.
			_, npubkey := QuickKey()
			ngrant, err := c.Grant(&Grant{Authorization: npubkey, Permission: UserFull()})
			Expect(err).To(Succeed())
			Expect(ngrant.Authorization).To(Equal(npubkey))
		})
	})

	Describe("Revoke", func() {
		It("should fail without credentials", func() {
			notary := New("", nil, NewDirectory(testingx.TempDir()))
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
			ngrant, err := c.Grant(&Grant{Authorization: npubkey, Permission: UserFull()})
			Expect(err).To(Succeed())
			Expect(ngrant.Authorization).To(Equal(npubkey))
			nrevoked, err := c.Revoke(ngrant.Fingerprint)
			Expect(err).To(Succeed())
			Expect(proto.Equal(nrevoked, ngrant)).To(BeTrue())
		})
	})
})
