package rsax

import (
	"crypto/md5"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

// these tests are just to test the functionality, not the actual randomness.
func TestAutoDeterministic(t *testing.T) {
	for _, c := range []struct {
		name     string
		seed     string
		bits     int
		expected string
	}{
		{"example 1", "helloworld", 4096, "7e6f446514043e118991e2407ce46b64"},
	} {
		t.Run(c.name, func(t *testing.T) {
			pkey, err := Deterministic([]byte(c.seed), c.bits)
			require.NoError(t, err)
			digest := md5.Sum(pkey)
			require.Equal(t, c.expected, hex.EncodeToString(digest[:]))
		})
	}
}
