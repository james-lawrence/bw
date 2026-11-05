package notary_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/sshx"
	. "github.com/james-lawrence/bw/notary"
)

func TestFileStorageReload(t *testing.T) {
	t.Run("should reload authorizations after repeated atomic replacement", func(t *testing.T) {
		root := t.TempDir()
		src := t.TempDir()
		cfg := filepath.Join(src, "notary.yml")
		require.NoError(t, os.WriteFile(cfg, []byte("notary: {}\n"), 0600))

		c, err := NewFromFile(root, cfg)
		require.NoError(t, err)

		newkey := func(comment string) (string, []byte) {
			s, err := QuickSigner(t)
			require.NoError(t, err)
			fp, pub, err := s.AutoSignerInfo()
			require.NoError(t, err)
			return fp, sshx.Comment(pub, comment)
		}

		fp1, pub1 := newkey("test1")
		fp2, pub2 := newkey("test2")

		swap := func(content []byte) {
			tmp := filepath.Join(src, "authorized_keys")
			require.NoError(t, os.WriteFile(tmp, content, 0600))
			require.NoError(t, CloneAuthorizationFile(tmp, filepath.Join(root, bw.AuthKeysFile)))
		}

		swap(pub1)
		require.Eventually(t, func() bool { _, err := c.Lookup(fp1); return err == nil }, 2*time.Second, 50*time.Millisecond)

		swap(pub2)
		require.Eventually(t, func() bool { _, err := c.Lookup(fp2); return err == nil }, 2*time.Second, 50*time.Millisecond)
		require.Eventually(t, func() bool { _, err := c.Lookup(fp1); return err != nil }, 2*time.Second, 50*time.Millisecond)
	})
}
