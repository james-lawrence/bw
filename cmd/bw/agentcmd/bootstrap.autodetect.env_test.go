package agentcmd_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/james-lawrence/bw/cmd/bw/agentcmd"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/stretchr/testify/require"
)

func TestBootstrapEnv(t *testing.T) {
	loopback := func(ip byte, port int) *net.TCPAddr {
		return &net.TCPAddr{IP: net.IPv4(127, 0, 0, ip), Port: port}
	}

	run := func(t *testing.T, existing string, cmd agentcmd.CmdBootstrapEnv) string {
		cmd.Output = filepath.Join(t.TempDir(), "agent.env")
		if existing != "" {
			require.NoError(t, os.WriteFile(cmd.Output, []byte(existing), 0600))
		}
		require.NoError(t, cmd.Run(&cmdopts.Global{Context: context.Background()}))
		b, err := os.ReadFile(cmd.Output)
		require.NoError(t, err)
		return string(b)
	}

	t.Run("detect aws", func(t *testing.T) {
		yes := func(context.Context) bool { return true }
		no := func(context.Context) bool { return false }
		require.Equal(t, agentcmd.Detected{AWS: true}, agentcmd.Detect(context.Background(), agentcmd.Probes{AWS: yes, GCloud: no}))
	})

	t.Run("detect gcloud", func(t *testing.T) {
		yes := func(context.Context) bool { return true }
		no := func(context.Context) bool { return false }
		require.Equal(t, agentcmd.Detected{GCloud: true}, agentcmd.Detect(context.Background(), agentcmd.Probes{AWS: no, GCloud: yes}))
	})

	t.Run("detect nothing", func(t *testing.T) {
		no := func(context.Context) bool { return false }
		require.Equal(t, agentcmd.Detected{}, agentcmd.Detect(context.Background(), agentcmd.Probes{AWS: no, GCloud: no}))
	})

	t.Run("merge into empty", func(t *testing.T) {
		require.Equal(t, "A=1\n", agentcmd.MergeEnvironment("", "A=1"))
	})

	t.Run("merge appends new keys", func(t *testing.T) {
		require.Equal(t, "X=1\nA=1\n", agentcmd.MergeEnvironment("X=1\n", "A=1"))
	})

	t.Run("merge replaces existing keys in place", func(t *testing.T) {
		require.Equal(t, "A=1\nX=1\n", agentcmd.MergeEnvironment("A=0\nX=1\n", "A=1"))
	})

	t.Run("merge is idempotent", func(t *testing.T) {
		require.Equal(t, "A=1\n", agentcmd.MergeEnvironment("A=1\n", "A=1"))
	})

	t.Run("writes loopback alternate and preserves other settings", func(t *testing.T) {
		out := run(t, "X=1\n", agentcmd.CmdBootstrapEnv{Alternates: []*net.TCPAddr{loopback(1, 2000)}})
		require.Contains(t, out, "X=1\n")
		require.Contains(t, out, "BEARDED_WOOKIE_AGENT_P2P_ALTERNATES=127.0.0.1:2000\n")
	})

	t.Run("writes multiple alternates and the p2p port", func(t *testing.T) {
		out := run(t, "", agentcmd.CmdBootstrapEnv{Alternates: []*net.TCPAddr{loopback(1, 3000), loopback(2, 3000)}, P2PPort: 8443})
		require.Contains(t, out, "BEARDED_WOOKIE_AGENT_P2P_ALTERNATES=127.0.0.1:3000,127.0.0.2:3000\n")
		require.Contains(t, out, "BEARDED_WOOKIE_AGENT_CLUSTER_P2P_DISCOVERY_PORT=8443\n")
	})

	t.Run("p2p port is not written when unset", func(t *testing.T) {
		out := run(t, "", agentcmd.CmdBootstrapEnv{Alternates: []*net.TCPAddr{loopback(1, 2000)}})
		require.NotContains(t, out, "DISCOVERY_PORT")
	})
}
