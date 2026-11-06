package agentcmd_test

import (
	"testing"

	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/stretchr/testify/require"
)

func TestQuorum(t *testing.T) {
	boot(t)
	require.NoError(t, actl(t).Run("quorum", agenttestutil.Environment, "--insecure"))
}
