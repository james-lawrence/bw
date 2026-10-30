package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// the tests provide what they need, they don't inherit the BEARDED_WOOKIE_ configuration of the machine running them.
	environ := envx.Isolate(os.Environ(), "BEARDED_WOOKIE_*")
	os.Clearenv()
	for _, kv := range environ {
		k, v, _ := strings.Cut(kv, "=")
		_ = os.Setenv(k, v)
	}

	os.Exit(m.Run())
}

// parse builds the production parser against a fresh defaults config (bound to 10.0.0.1:2000) and returns the
// parsed command line and the defaults after the command hooks ran. Parse resolves flags, environment and hooks but
// doesn't run the command, so the daemon is never started.
func parse(t *testing.T, args ...string) (*cli, agent.Config, error) {
	t.Helper()

	defaults := agent.NewConfig(agent.ConfigOptionDefaultBind(net.IPv4(10, 0, 0, 1)))
	c := new(cli)
	_, err := newcmdroot(c, &defaults, kong.Exit(func(int) {})).Parse(args)

	return c, defaults, err
}

func TestAgentAddress(t *testing.T) {
	for name, tc := range map[string]struct {
		args        []string
		bind        string
		advertised  string
		alternates  []string
		description string
	}{
		"defaults":                          {args: []string{"agent", "runtime"}, bind: "10.0.0.1:2000", advertised: "10.0.0.1:2000"},
		"bind flag":                         {args: []string{"agent", "runtime", "--agent-address=127.0.0.3:2000"}, bind: "127.0.0.3:2000", advertised: "127.0.0.3:2000"},
		"advertised flag":                   {args: []string{"agent", "runtime", "--agent-address=127.0.0.3:2000", "--agent-address-advertised=203.0.113.9:2000"}, bind: "127.0.0.3:2000", advertised: "203.0.113.9:2000"},
		"alternates comma separated":        {args: []string{"agent", "runtime", "--agent-address-bindings=127.0.0.1:2001,127.0.0.1:2002"}, bind: "10.0.0.1:2000", advertised: "10.0.0.1:2000", alternates: []string{"127.0.0.1:2001", "127.0.0.1:2002"}},
		"alternates space separated":        {args: []string{"agent", "runtime", "--agent-address-bindings=127.0.0.1:2001 127.0.0.1:2002"}, bind: "10.0.0.1:2000", advertised: "10.0.0.1:2000", alternates: []string{"127.0.0.1:2001", "127.0.0.1:2002"}},
		"coordinator applies the same":      {args: []string{"agent", "coordinator", "--agent-address=127.0.0.3:2000"}, bind: "127.0.0.3:2000", advertised: "127.0.0.3:2000"},
		"runtime is the default subcommand": {args: []string{"agent", "--agent-address=127.0.0.3:2000", "--agent-address-advertised=203.0.113.9:2000"}, bind: "127.0.0.3:2000", advertised: "203.0.113.9:2000"},
	} {
		t.Run(name, func(t *testing.T) {
			_, config, err := parse(t, tc.args...)
			require.NoError(t, err)
			require.Equal(t, tc.bind, config.P2PBind.String())
			require.Equal(t, tc.advertised, config.P2PAdvertised.String())

			alternates := make([]string, 0, len(config.AlternateBinds))
			for _, a := range config.AlternateBinds {
				alternates = append(alternates, a.String())
			}
			require.Equal(t, tc.alternates, append([]string(nil), alternates...))
		})
	}

	t.Run("an invalid address is an error", func(t *testing.T) {
		_, _, err := parse(t, "agent", "runtime", "--agent-address=not an address")
		require.Error(t, err)
	})
}

// each environment variable has to reach the flag it is bound to, kong reads these from the process environment.
func TestAgentEnvironment(t *testing.T) {
	t.Run("bind", func(t *testing.T) {
		t.Setenv("BEARDED_WOOKIE_AGENT_P2P_BIND", "127.0.0.3:2000")
		_, config, err := parse(t, "agent", "runtime")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.3:2000", config.P2PBind.String())
	})

	t.Run("advertised", func(t *testing.T) {
		t.Setenv("BEARDED_WOOKIE_AGENT_P2P_ADVERTISED", "203.0.113.9:2000")
		_, config, err := parse(t, "agent", "runtime")
		require.NoError(t, err)
		require.Equal(t, "203.0.113.9:2000", config.P2PAdvertised.String())
	})

	t.Run("alternates", func(t *testing.T) {
		t.Setenv("BEARDED_WOOKIE_AGENT_P2P_ALTERNATES", "127.0.0.1:2001,127.0.0.1:2002")
		_, config, err := parse(t, "agent", "runtime")
		require.NoError(t, err)
		require.Len(t, config.AlternateBinds, 2)
	})

	t.Run("bootstrap", func(t *testing.T) {
		t.Setenv("BEARDED_WOOKIE_AGENT_BOOTSTRAP", "127.0.0.2:2000")
		c, _, err := parse(t, "agent", "runtime")
		require.NoError(t, err)
		require.Len(t, c.Agent.Runtime.Bootstrap, 1)
		require.Equal(t, "127.0.0.2:2000", c.Agent.Runtime.Bootstrap[0].String())
	})

	t.Run("flag beats the environment", func(t *testing.T) {
		t.Setenv("BEARDED_WOOKIE_AGENT_P2P_BIND", "127.0.0.9:2000")
		_, config, err := parse(t, "agent", "runtime", "--agent-address=127.0.0.3:2000")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.3:2000", config.P2PBind.String())
	})
}

// the default subcommand (default:"withargs") accepts the flags of runtime, and resolves with none.
func TestAgentDefaultsToRuntime(t *testing.T) {
	t.Run("environment", func(t *testing.T) {
		t.Setenv("BEARDED_WOOKIE_AGENT_P2P_BIND", "127.0.0.3:2000")
		_, config, err := parse(t, "agent")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.3:2000", config.P2PBind.String())
	})

	t.Run("flags are those of runtime", func(t *testing.T) {
		c, config, err := parse(t, "agent", "--agent-address=127.0.0.3:2000", "--bootstrap-static-addresses=127.0.0.2:2000", "--no-bootstrap-p2p-enable")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.3:2000", config.P2PBind.String())
		require.Len(t, c.Agent.Runtime.Bootstrap, 1)
		require.False(t, c.Agent.Runtime.P2PEnabled)
	})

	t.Run("explicit runtime is unchanged", func(t *testing.T) {
		_, config, err := parse(t, "agent", "runtime", "--agent-address=127.0.0.3:2000")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.3:2000", config.P2PBind.String())
	})

	t.Run("the other subcommands are still reachable", func(t *testing.T) {
		for _, args := range [][]string{{"agent", "coordinator"}, {"agent", "quorum"}, {"agent", "quorum-state", "/tmp/quorum.db"}} {
			_, _, err := parse(t, args...)
			require.NoError(t, err, args)
		}
	})
}

func TestAgentBootstrap(t *testing.T) {
	c, _, err := parse(t, "agent", "runtime")
	require.NoError(t, err)
	require.Empty(t, c.Agent.Runtime.Bootstrap)
	require.True(t, c.Agent.Runtime.P2PEnabled)

	c, _, err = parse(t, "agent", "runtime", "--bootstrap-static-addresses=127.0.0.2:2000", "--no-bootstrap-p2p-enable")
	require.NoError(t, err)
	require.Len(t, c.Agent.Runtime.Bootstrap, 1)
	require.False(t, c.Agent.Runtime.P2PEnabled)
}

// the integration suite starts these agents from their environment files, if the wiring regresses the agents
// bind the same address and the cluster never forms.
func TestLinuxDevelopmentTopology(t *testing.T) {
	seen := map[string]int{}

	for i := 1; i <= 4; i++ {
		t.Run(fmt.Sprintf("agent%d", i), func(t *testing.T) {
			path := fmt.Sprintf("../../.test/linux-development/agent%d/agent.env", i)
			env, err := godotenv.Read(path)
			require.NoError(t, err)
			for k, v := range env {
				t.Setenv(k, v)
			}

			c, config, err := parse(t, "agent", "runtime")
			require.NoError(t, err)
			require.Equal(t, env["BEARDED_WOOKIE_AGENT_P2P_BIND"], config.P2PBind.String(), path)

			_, port, err := net.SplitHostPort(env["BEARDED_WOOKIE_AGENT_BOOTSTRAP"])
			require.NoError(t, err)
			require.Len(t, c.Agent.Runtime.Bootstrap, 1, path)
			require.Equal(t, port, strconv.Itoa(c.Agent.Runtime.Bootstrap[0].Port), path)

			if prev, dup := seen[config.P2PBind.String()]; dup {
				t.Fatalf("agent%d and agent%d both bind %s", prev, i, config.P2PBind)
			}
			seen[config.P2PBind.String()] = i
		})
	}
}

// the gdx commands default to the socket the agent serves.
func TestGDX(t *testing.T) {
	t.Setenv("RUNTIME_DIRECTORY", "/run/bearded-wookie/agent1")

	for name, args := range map[string][]string{
		"goroutines": {"gdx", "goroutines"},
		"prof":       {"gdx", "prof", "heap"},
		"trace":      {"gdx", "trace"},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := parse(t, args...)
			require.NoError(t, err)
		})
	}

	t.Run("socket defaults to the agent's runtime directory", func(t *testing.T) {
		require.Equal(t, filepath.Join("/run/bearded-wookie/agent1", "gdx.socket"), gdxSocket())
		c, _, err := parse(t, "gdx", "goroutines")
		require.NoError(t, err)
		require.Equal(t, gdxSocket(), c.GDX.Goroutines.Socket)
	})

	t.Run("socket can be overridden", func(t *testing.T) {
		c, _, err := parse(t, "gdx", "goroutines", "--socket=/tmp/other.socket")
		require.NoError(t, err)
		require.Equal(t, "/tmp/other.socket", c.GDX.Goroutines.Socket)
	})
}
