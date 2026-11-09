package deploy_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/stretchr/testify/require"
)

func TestCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("boots an agent, skipping in short mode")
	}

	// cancelling a deploy must return as soon as the agent acknowledges the cancel,
	// previously it blocked until the process was killed.
	t.Run("returns once the agent acknowledges the cancel", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		// small deterministic key, generating a full size key dominates the time of the test.
		seed := rsax.DeterministcSeed("bw.test", "cancel", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })

		require.NoError(t, agenttestutil.Agent(t, client, addr))
		require.NoError(t, agenttestutil.Client(t, client, addr))
		require.NoError(t, agenttestutil.WaitQuorum(context.Background()))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := make(chan error, 1)
		go func() {
			result <- deploy.Cancel(&deploy.Context{
				Environment: agenttestutil.Environment,
				Insecure:    true,
				Context:     ctx,
				CancelFunc:  cancel,
				WaitGroup:   &sync.WaitGroup{},
			})
		}()

		select {
		case err := <-result:
			require.NoError(t, err)
		case <-time.After(20 * time.Second):
			t.Fatal("deploy cancel did not return after the cancel was acknowledged")
		}
	})

	// every node of the cluster must stop its deploy, not just the first one reached.
	t.Run("cancels the deploy on every node", func(t *testing.T) {
		free := func() *net.TCPAddr {
			l, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer l.Close()
			return l.Addr().(*net.TCPAddr)
		}
		addrs := []*net.TCPAddr{free(), free()}

		seed := rsax.DeterministcSeed("bw.test", "cancel", "multi", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })
		nodes := func(i int) agent.ConfigOption {
			return agent.ConfigOptionCompose(func(c *agent.Config) { c.MinimumNodes = len(addrs) }, agent.ConfigOptionName(fmt.Sprintf("node%d", i)))
		}

		// agents block until the cluster has its minimum nodes, so they must start together.
		started := make(chan error, len(addrs))
		go func() { started <- agenttestutil.Agent(t, client, addrs[0], nodes(0)) }()
		for i, a := range addrs[1:] {
			go func() { started <- agenttestutil.AgentJoin(t, client, a, addrs[0], nodes(i+1)) }()
		}
		for range addrs {
			require.NoError(t, <-started)
		}
		require.NoError(t, agenttestutil.Client(t, client, addrs[0]))
		require.NoError(t, agenttestutil.WaitNodes(context.Background(), len(addrs)))

		// a directive that outlives the test unless it is cancelled. every node records the pid of its process.
		pids := filepath.Join(t.TempDir(), "pids")
		remote := filepath.Join(bw.DefaultDeployspaceDir, ".remote")
		require.NoError(t, os.MkdirAll(remote, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(remote, "00_block.bwcmd"), []byte("- command: sh -c 'echo started >> "+pids+"; sleep 987'\n"), 0600))

		dctx, dcancel := context.WithCancel(context.Background())
		defer dcancel()
		go deploy.Into(&deploy.Context{
			Environment: agenttestutil.Environment,
			Insecure:    true,
			Attempts:    1,
			Heartbeat:   10 * time.Second,
			Filter:      deployment.AlwaysMatch,
			Context:     dctx,
			CancelFunc:  dcancel,
			WaitGroup:   &sync.WaitGroup{},
		})

		begun := func() int {
			raw, _ := os.ReadFile(pids)
			return len(strings.Fields(string(raw)))
		}
		// the processes left behind by the deploys, the sleep is unique to this test.
		running := func() int {
			out, _ := exec.Command("pgrep", "-fc", "sleep 987").Output()
			n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
			return n
		}

		require.Eventually(t, func() bool { return begun() == len(addrs) && running() >= len(addrs) }, 30*time.Second, 100*time.Millisecond, "every node should be deploying")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		require.NoError(t, deploy.Cancel(&deploy.Context{
			Environment: agenttestutil.Environment,
			Insecure:    true,
			Context:     ctx,
			CancelFunc:  cancel,
			WaitGroup:   &sync.WaitGroup{},
		}))

		require.Eventually(t, func() bool { return running() == 0 }, 10*time.Second, 100*time.Millisecond, "every node should have cancelled its deploy")
	})
}
