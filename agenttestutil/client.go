package agenttestutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/internal/fsx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/vcsinfo"
	"google.golang.org/grpc"
)

// Client writes the configuration for Environment, pointing at the agent bound to addr,
// into the workspace created by Agent. it must be called after Agent, and doesn't wait for the agent
// to be ready, see WaitForQuorum. the options are applied last so tests can override anything.
func Client(t *testing.T, addr *net.TCPAddr, options ...agent.ConfigClientOption) error {
	t.Helper()

	workspace, err := os.Getwd()
	if err != nil {
		return err
	}

	envdir := filepath.Join(workspace, bw.DefaultDeployspaceConfigDir, Environment)
	if err = fsx.MkDirs(0700, envdir); err != nil {
		return err
	}

	// CCOptionAddress isn't used, it forces the default port.
	client := agent.NewConfigClient(
		agent.ExampleConfigClient(agent.CCOptionTLSConfig(Environment), agent.CCOptionConcurrency(1)),
		append([]agent.ConfigClientOption{
			func(c *agent.ConfigClient) {
				c.Address = addr.String()
				c.ServerName = "localhost"
				c.Credentials.Mode = "disabled"
				c.Deployment.Prompt = ""
			},
		}, options...)...,
	)

	return WriteClient(filepath.Join(envdir, bw.DefaultClientConfig), client)
}

// WaitQuorum blocks until the agent has joined the quorum and has a leader.
// the agent accepts connections before then, but rejects the operations that require the quorum.
func WaitQuorum(parent context.Context) error {
	ctx, done := context.WithTimeout(parent, 30*time.Second)
	defer done()

	config, err := commandutils.LoadConfiguration(ctx, Environment, agent.CCOptionInsecure(true))
	if err != nil {
		return err
	}

	ss, err := notary.NewAutoSigner(vcsinfo.CurrentUserDisplay(config.WorkDir()))
	if err != nil {
		return err
	}

	d, c, err := daemons.ConnectClientUntilSuccess(ctx, config, ss, grpc.WithPerRPCCredentials(ss))
	if err != nil {
		return err
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	for {
		if hasLeader(ctx, qd) {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("agent did not join the quorum: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func hasLeader(ctx context.Context, qd dialers.Quorum) bool {
	conn, err := qd.DialContext(ctx)
	if err != nil {
		return false
	}
	defer conn.Close()

	info, err := agent.NewQuorumClient(conn).Info(ctx, &agent.InfoRequest{})
	return err == nil && info.Leader != nil
}
