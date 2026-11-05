package bw_test

import (
	"github.com/stretchr/testify/require"
	"testing"

	. "github.com/james-lawrence/bw"
)

func TestBw(t *testing.T) {
	t.Run("SimpleGenerateID", func(t *testing.T) {
		t.Run("16 bytes", func(t *testing.T) {
			id := MustGenerateID()
			require.Len(t, id[:], 16)
		})

		t.Run("should generate different ids", func(t *testing.T) {
			var (
				cache = make(map[string]struct{})
			)

			for range 100 {
				id := MustGenerateID()
				_, found := cache[id.String()]
				require.False(t, found)
				cache[id.String()] = struct{}{}
			}
		})
	})
}
