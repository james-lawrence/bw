package agenttestutil

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/bw/agentcmd"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/cmdtestx"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/fsx"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/userx"
	"github.com/james-lawrence/bw/notary"
)

// Environment is the name of the environment the client of the test agent is configured for.
const Environment = "e2e"

// Agent starts a single node agent bound to addr, running in-process until the test completes.
//
// everything it touches lives in a directory created for the test, and is removed when it completes.
// the client key is authorized by the agent, and the current directory is changed to the workspace Client writes
// the configuration for Environment into.
// Agent returns once the services are started, use Client to configure the client and wait for the quorum.
//
// the agent reads parts of its configuration from the process environment, and the logger is global,
// so tests using Agent can't be parallel.
func Agent(t *testing.T, addr *net.TCPAddr, options ...agent.ConfigOption) error {
	t.Helper()

	// not t.TempDir(), the unix sockets the agent creates have a limited path length that
	// the name of the test could exceed.
	dir, err := os.MkdirTemp("", "bw.test.agent.*")
	if err != nil {
		return err
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	var (
		xdg       = filepath.Join(dir, "xdg")
		workspace = filepath.Join(dir, "workspace")
		root      = filepath.Join(dir, "a")
		agentcfg  = filepath.Join(dir, "agent.yml")
	)

	if err = fsx.MkDirs(
		0700,
		filepath.Join(xdg, "bearded-wookie"), // must exist or the user's home is used.
		filepath.Join(dir, "cache"),
		filepath.Join(dir, "run"),
		filepath.Join(root, "authorizations"),
		workspace,
	); err != nil {
		return err
	}

	// the daemon and the client read these directly from the process rather than through the command line.
	environ, err := envx.Build().
		Var("XDG_CONFIG_HOME", xdg).
		Var("XDG_CACHE_HOME", filepath.Join(dir, "cache")).
		Var("RUNTIME_DIRECTORY", filepath.Join(dir, "run")).
		Environ()
	if err != nil {
		return err
	}

	for _, kv := range environ {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}

	t.Chdir(workspace)

	// the agent only accepts clients it has authorized. the key is small and deterministic,
	// generating a full size key dominates the time of the test.
	seed := rsax.DeterministcSeed("bw.test.agent", addr.String(), "client")
	if _, err = notary.NewDeterministicSigner(func() ([]byte, error) {
		return rsax.Deterministic(seed, rsax.UnsafeBits())
	}, "test"); err != nil {
		return err
	}

	// the agent generates its p2p key on startup unless one exists. same reasoning as the client key.
	p2pkey, err := rsax.Deterministic(rsax.DeterministcSeed("bw.test.agent", addr.String(), "p2p"), rsax.UnsafeBits())
	if err != nil {
		return err
	}

	if err = os.WriteFile(userx.DefaultRuntimeDirectory(bw.DefaultAgentNotaryKey), p2pkey, 0600); err != nil {
		return err
	}

	_, pub, err := notary.AutoSignerInfo()
	if err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(root, "authorizations", "bw.auth.keys"), pub, 0600); err != nil {
		return err
	}

	// the options are applied last so tests can override anything.
	// credentials are disabled so the agent doesn't try to acquire a certificate from an ACME provider.
	defaults := agent.NewConfig(append([]agent.ConfigOption{
		agent.ConfigOptionP2P(addr),
		agent.ConfigOptionAdvertised(addr),
		func(c *agent.Config) {
			c.Root = root
			c.MinimumNodes = 1
			c.ServerName = "localhost"
			c.Credentials.Mode = "disabled"
		},
	}, options...)...).EnsureDefaults()

	if err = WriteAgent(agentcfg, defaults); err != nil {
		return err
	}

	// the daemon sets the prefix of the global logger.
	prefix := log.Prefix()
	t.Cleanup(func() { log.SetPrefix(prefix) })

	err = cmdtestx.New(t, &agentcmd.CmdDaemon{}).
		Options(cmdopts.Vars(defaults), cmdopts.Mappers(), kong.Bind(&defaults)).
		Run("runtime", "--agent-config="+agentcfg, "--agent-address="+addr.String())
	return err
}
