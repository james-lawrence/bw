package notary_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/james-lawrence/bw/internal/testingx"

	. "github.com/james-lawrence/bw/notary"
)

func TestDirectory(t *testing.T) {
	t.Run("should be able to read/write/delete grants", func(t *testing.T) {
		s := NewDirectory(
			testingx.TempDir(t),
		)

		g1, err := s.Insert(&Grant{
			Authorization: []byte{0},
		})
		require.NoError(t, err)

		g1f, err := s.Lookup(g1.Fingerprint)
		require.NoError(t, err)
		require.True(t, proto.Equal(g1f, g1))

		g1d, err := s.Delete(g1)
		require.NoError(t, err)
		require.True(t, proto.Equal(g1d, g1))
	})

	t.Run("should be able able to overwrite a grant", func(t *testing.T) {
		s := NewDirectory(
			testingx.TempDir(t),
		)

		g1, err := s.Insert(&Grant{
			Authorization: []byte{0},
		})
		require.NoError(t, err)
		g1u, err := s.Insert(&Grant{
			Authorization: []byte{0},
		})
		require.NoError(t, err)

		require.True(t, proto.Equal(g1u, g1))
		g1f, err := s.Lookup(g1.Fingerprint)
		require.NoError(t, err)
		require.True(t, proto.Equal(g1f, g1))
		require.True(t, proto.Equal(g1f, g1u))
	})
}
