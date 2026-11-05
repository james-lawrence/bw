package deploy_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/vcsinfo"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestRedeploy(t *testing.T) {
	if testing.Short() {
		t.Skip("boots an agent, skipping in short mode")
	}

	// redeploying must run the archive of a previous deploy again, and return once the agent completes it.
	t.Run("runs the archive of a previous deploy again", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		seed := rsax.DeterministcSeed("bw.test", "redeploy", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })

		require.NoError(t, agenttestutil.Agent(t, client, addr))
		require.NoError(t, agenttestutil.Client(t, client, addr))
		require.NoError(t, agenttestutil.WaitQuorum(context.Background()))

		marker := filepath.Join(t.TempDir(), "marker")
		remote := filepath.Join(bw.DefaultDeployspaceDir, ".remote")
		require.NoError(t, os.MkdirAll(remote, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(remote, "00_marker.bwcmd"), []byte("- command: touch "+marker+"\n"), 0600))

		ictx, icancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer icancel()

		require.NoError(t, deploy.Into(&deploy.Context{
			Environment: agenttestutil.Environment,
			Insecure:    true,
			Attempts:    1,
			Heartbeat:   10 * time.Second,
			Filter:      deployment.AlwaysMatch,
			Context:     ictx,
			CancelFunc:  icancel,
			WaitGroup:   &sync.WaitGroup{},
		}))
		require.FileExists(t, marker)
		require.NoError(t, os.Remove(marker))

		// locate the deployment that was just run.
		config, err := commandutils.LoadConfiguration(context.Background(), agenttestutil.Environment, agent.CCOptionInsecure(true))
		require.NoError(t, err)

		ss, err := notary.NewAutoSigner(vcsinfo.CurrentUserDisplay(config.WorkDir()))
		require.NoError(t, err)

		d, c, err := daemons.ConnectClientUntilSuccess(context.Background(), config, ss, grpc.WithPerRPCCredentials(ss))
		require.NoError(t, err)

		located, err := agentutil.LocateDeployment(
			cluster.New(commandutils.NewClientPeer(), c),
			dialers.NewQuorum(c, d.Defaults()...),
			func(d *agent.Deploy) bool { return d.Archive != nil },
		)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := make(chan error, 1)
		go func() {
			result <- deploy.Redeploy(&deploy.Context{
				Environment: agenttestutil.Environment,
				Insecure:    true,
				Attempts:    1,
				Heartbeat:   10 * time.Second,
				Filter:      deployment.AlwaysMatch,
				Context:     ctx,
				CancelFunc:  cancel,
				WaitGroup:   &sync.WaitGroup{},
			}, bw.RandomID(located.Archive.DeploymentID).String())
		}()

		select {
		case err := <-result:
			require.NoError(t, err)
		case <-time.After(20 * time.Second):
			t.Fatal("redeploy did not return after the agent completed the deploy")
		}
		require.FileExists(t, marker)
	})
}
