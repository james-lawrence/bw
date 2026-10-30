package commandutils_test

import (
	"net"
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/stretchr/testify/require"
)

// the agent merges its configuration file over the config built from the flags,
// the file must not undo the addresses given on the command line.
func TestLoadAgentConfigKeepsTheBindAddresses(t *testing.T) {
	t.Setenv("CACHE_DIRECTORY", t.TempDir())
	t.Setenv("STATE_DIRECTORY", t.TempDir())
	t.Setenv("BEARDED_WOOKIE_AGENT_SERVERNAME", "localhost")

	bind := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 3), Port: 2000}
	proto := agent.NewConfig(agent.ConfigOptionP2P(bind), agent.ConfigOptionAdvertised(bind))

	config, err := commandutils.LoadAgentConfig("../../.test/linux-development/agent3/agent.config", proto)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.3:2000", config.P2PBind.String())
	require.Equal(t, "127.0.0.3:2000", config.P2PAdvertised.String())
	require.Equal(t, "localhost", config.ServerName, "the file is applied")
	require.Equal(t, 3, config.MinimumNodes, "the file is applied")
}
