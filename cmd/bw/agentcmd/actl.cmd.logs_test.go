package agentcmd_test

import (
	"testing"

	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/stretchr/testify/require"
)

// a fresh agent has no deployment to retrieve logs for.
func TestLogsWithoutDeployment(t *testing.T) {
	boot(t)
	require.Error(t, actl(t).Run("logs", agenttestutil.Environment, "--insecure"))
}
