package agentcmd_test

import (
	"testing"

	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/stretchr/testify/require"
)

func TestNodes(t *testing.T) {
	boot(t)
	require.NoError(t, actl(t).Run("nodes", agenttestutil.Environment, "--insecure"))
}
