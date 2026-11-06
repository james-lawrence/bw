package agentcmd_test

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/cmd/bw/agentcmd"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/cmdtestx"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/stretchr/testify/require"
)

// boot starts an in-process agent and configures the client environment for it, returning its address.
func boot(t *testing.T) *net.TCPAddr {
	t.Helper()

	if testing.Short() {
		t.Skip("boots an agent, skipping in short mode")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().(*net.TCPAddr)
	require.NoError(t, l.Close())

	seed := rsax.DeterministcSeed("bw.test", "actl", t.Name())
	client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })

	require.NoError(t, agenttestutil.Agent(t, client, addr))
	require.NoError(t, agenttestutil.Client(t, client, addr))
	require.NoError(t, agenttestutil.WaitQuorum(context.Background()))

	return addr
}

// actl returns a harness running the actl commands.
func actl(t *testing.T) *cmdtestx.Harness {
	t.Helper()

	return cmdtestx.New(t, &agentcmd.CmdControl{}).Options(cmdopts.Vars(agent.NewConfig()), cmdopts.Mappers())
}
