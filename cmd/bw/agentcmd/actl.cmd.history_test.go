package agentcmd_test

import (
	"testing"

	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/stretchr/testify/require"
)

func TestHistory(t *testing.T) {
	boot(t)
	require.NoError(t, actl(t).Run("history", agenttestutil.Environment, "--insecure"))
}

func TestHistoryJSONL(t *testing.T) {
	boot(t)
	require.NoError(t, actl(t).Run("history", agenttestutil.Environment, "--insecure", "--format-jsonl"))
}
