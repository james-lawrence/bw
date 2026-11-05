package notary_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	. "github.com/james-lawrence/bw/notary"

	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/internal/testingx"
)

func TestNotary(t *testing.T) {
	t.Run("Grant", func(t *testing.T) {
		t.Run("should fail without credentials", func(t *testing.T) {
			notary := New("", nil, NewDirectory(testingx.TempDir(t)))
			conn, server := testingx.NewGRPCServer(t, func(s *grpc.Server) {
				notary.Bind(s)
			})
			defer testingx.GRPCCleanup(t, conn, server)

			c := NewClient(NewStaticDialer(conn))
			_, err := c.Grant(&Grant{})
			require.EqualError(t, err, "rpc error: code = PermissionDenied desc = invalid credentials")
		})

		t.Run("should succeed with credentials", func(t *testing.T) {
			c, cleanup := QuickService(t)
			defer cleanup()

			// generate a new key to add.
			_, npubkey := QuickKey(t)
			ngrant, err := c.Grant(&Grant{Authorization: npubkey, Permission: UserFull()})
			require.NoError(t, err)
			require.Equal(t, npubkey, ngrant.Authorization)
		})
	})

	t.Run("Revoke", func(t *testing.T) {
		t.Run("should fail without credentials", func(t *testing.T) {
			notary := New("", nil, NewDirectory(testingx.TempDir(t)))
			conn, server := testingx.NewGRPCServer(t, func(s *grpc.Server) {
				notary.Bind(s)
			})
			defer testingx.GRPCCleanup(t, conn, server)

			c := NewClient(NewStaticDialer(conn))
			_, err := c.Revoke("")
			require.EqualError(t, err, "rpc error: code = PermissionDenied desc = invalid credentials")
		})

		t.Run("should succeed with credentials", func(t *testing.T) {
			c, cleanup := QuickService(t)
			defer cleanup()

			// generate a new key to add.
			_, npubkey := QuickKey(t)
			ngrant, err := c.Grant(&Grant{Authorization: npubkey, Permission: UserFull()})
			require.NoError(t, err)
			require.Equal(t, npubkey, ngrant.Authorization)
			nrevoked, err := c.Revoke(ngrant.Fingerprint)
			require.NoError(t, err)
			require.True(t, proto.Equal(nrevoked, ngrant))
		})
	})
}
