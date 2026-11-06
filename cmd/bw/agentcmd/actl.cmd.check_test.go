package agentcmd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheck(t *testing.T) {
	t.Run("address is required", func(t *testing.T) {
		require.Error(t, actl(t).Run("check"))
	})

	t.Run("reachable agent", func(t *testing.T) {
		addr := boot(t)
		require.NoError(t, actl(t).Run("check", "--insecure", addr.String()))
	})
}
