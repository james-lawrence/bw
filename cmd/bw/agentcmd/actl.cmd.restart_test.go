package agentcmd_test

import (
	"testing"

	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/stretchr/testify/require"
)

// without --force the command only lists the agents, it must not restart them.
func TestRestartWithoutForceDoesNothing(t *testing.T) {
	boot(t)
	require.NoError(t, actl(t).Run("restart", agenttestutil.Environment, "--insecure"))
}
