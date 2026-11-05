package notary_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw/internal/sshx"
	. "github.com/james-lawrence/bw/notary"
)

func TestReplaceAuthorizedKey(t *testing.T) {
	t.Run("should allow repeated replace calls without change", func(t *testing.T) {
		auths, err := os.CreateTemp(t.TempDir(), "auths.tmp")
		require.NoError(t, err)

		s1, err := QuickSigner(t)
		require.NoError(t, err)
		fp1, pubk1, err := s1.AutoSignerInfo()
		require.NoError(t, err)
		require.NoError(t, ReplaceAuthorizedKey(auths.Name(), fp1, sshx.Comment(pubk1, "test1")))

		s2, err := QuickSigner(t)
		require.NoError(t, err)
		fp2, pubk2, err := s2.AutoSignerInfo()
		require.NoError(t, err)
		require.NoError(t, ReplaceAuthorizedKey(auths.Name(), fp2, sshx.Comment(pubk2, "test2")))
		require.NoError(t, ReplaceAuthorizedKey(auths.Name(), fp2, sshx.Comment(pubk2, "test2")))

		content, err := os.ReadFile(auths.Name())
		require.NoError(t, err)
		require.Equal(t, string(bytes.Join(
			[][]byte{
				sshx.Comment(pubk1, "test1"),
				sshx.Comment(pubk2, "test2"),
			},
			[]byte{},
		)), string(content))
	})
}
