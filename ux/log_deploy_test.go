package ux_test

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/stretchr/testify/require"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/contextx"
	"github.com/james-lawrence/bw/internal/errorsx"
	. "github.com/james-lawrence/bw/ux"
)

func TestLogDeploy(t *testing.T) {
	t.Run("should process every message", func(t *testing.T) {
		check := func(t *testing.T, failure error, messages ...*agent.Message) {
			buf := make(chan *agent.Message, len(messages))
			for _, m := range messages {
				buf <- m
			}
			ctx := contextx.NewWaitGroup(context.Background())
			ctx, failed := context.WithCancelCause(ctx)
			Deploy(ctx, failed, nil, buf, agent.NewPeer("local"))
			require.Equal(t, 0, len(buf))
			if failure != nil {
				require.ErrorIs(t, context.Cause(ctx), failure)
			} else {
				require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))
			}
		}
		t.Run("successful deploy", func(t *testing.T) {
			check(t, error(nil), agent.LogInfo(agent.NewPeer("node1"), "hello world"), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}), agent.LogInfo(agent.NewPeer("node1"), "info message"), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}))
		})
		t.Run("failed deploy", func(t *testing.T) {
			check(t, errorsx.String("deploy failed"), agent.LogInfo(agent.NewPeer("node1"), "hello world"), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}), agent.LogInfo(agent.NewPeer("node1"), "info message"), agent.DeployEvent(agent.NewPeer("node1"), &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}, Error: "boom"}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}))
		})
		t.Run("automatic restart deploy", func(t *testing.T) {
			check(t, error(nil), agent.LogInfo(agent.NewPeer("node1"), "hello world"), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}), agent.LogInfo(agent.NewPeer("node1"), "info message"), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Restart, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}), agent.LogInfo(agent.NewPeer("node1"), "info message"), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Cancel, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}), agent.LogInfo(agent.NewPeer("node1"), "info message"), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}))
		})
		t.Run("automatic restart after a node failure", func(t *testing.T) {
			check(t, error(nil), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}), agent.DeployEvent(agent.NewPeer("node1"), &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}, Error: "boom"}), agent.NewDeployCommand(agent.NewPeer("node1"), agent.DeployCommandRestart()), agent.NewDeployCommand(agent.NewPeer("node1"), agent.DeployCommandCancel("")), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}))
		})
	})

	t.Run("should only finish on commands for the monitored deployment", func(t *testing.T) {
		check := func(t *testing.T, failure error, messages ...*agent.Message) {
			id := []byte("ours")
			monitored := new(atomic.Pointer[[]byte])
			monitored.Store(&id)

			buf := make(chan *agent.Message, len(messages))
			for _, m := range messages {
				buf <- m
			}
			ctx := contextx.NewWaitGroup(context.Background())
			ctx, failed := context.WithCancelCause(ctx)
			Deploy(ctx, failed, nil, buf, agent.NewPeer("local"), OptionDeployment(monitored))
			require.Equal(t, 0, len(buf))
			if failure != nil {
				require.ErrorIs(t, context.Cause(ctx), failure)
			} else {
				require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))
			}
		}
		t.Run("previous deploy completed", func(t *testing.T) {
			check(t, error(nil), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}))
		})
		t.Run("previous deploy cancelled", func(t *testing.T) {
			check(t, error(nil), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), agent.DeployCommandCancel("someone")), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}))
		})
		t.Run("previous deploy failed", func(t *testing.T) {
			check(t, error(nil), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}), agent.DeployEvent(agent.NewPeer("node1"), &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}, Error: "boom"}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}))
		})
		t.Run("cancelled", func(t *testing.T) {
			check(t, error(nil), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}), agent.NewDeployCommand(agent.NewPeer("node1"), agent.DeployCommandCancel("someone")))
		})
		t.Run("failed", func(t *testing.T) {
			check(t, errorsx.String("deploy failed"), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}), agent.DeployEvent(agent.NewPeer("node1"), &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}, Error: "boom"}), agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}))
		})
	})

	t.Run("should replay history when the last message is unknown", func(t *testing.T) {
		check := func(t *testing.T, monitored bool, finished bool, messages ...*agent.Message) {
			options := []Option{OptionStarted(time.Now().Add(-time.Hour))}
			if monitored {
				id := []byte("ours")
				deploymentID := new(atomic.Pointer[[]byte])
				deploymentID.Store(&id)
				options = append(options, OptionDeployment(deploymentID))
			}

			buf := make(chan *agent.Message, len(messages))
			for _, m := range messages {
				buf <- m
			}

			ctx, timeout := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer timeout()
			ctx = contextx.NewWaitGroup(ctx)
			ctx, failed := context.WithCancelCause(ctx)
			Deploy(ctx, failed, nil, buf, agent.NewPeer("local"), options...)
			if finished {
				require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))
			} else {
				require.ErrorIs(t, context.Cause(ctx), context.DeadlineExceeded)
			}
		}
		t.Run("first connect after the deploy completed", func(t *testing.T) {
			check(t, true, true, agent.NewLogHistoryFromMessages(
				agent.NewPeer("local"),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			))
		})
		t.Run("last message no longer in the history", func(t *testing.T) {
			check(t, true, true, agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}), agent.NewLogHistoryFromMessages(
				agent.NewPeer("local"),
				agent.PeersCompletedEvent(agent.NewPeer("node1"), 1),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			))
		})
		t.Run("history of only a previous deployment is ignored", func(t *testing.T) {
			check(t, true, false, agent.NewLogHistoryFromMessages(
				agent.NewPeer("local"),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			))
		})
		t.Run("first connect without a monitored deployment ignores the history", func(t *testing.T) {
			check(t, false, false, agent.NewLogHistoryFromMessages(
				agent.NewPeer("local"),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			))
		})
	})

	t.Run("should not replay history from before the client started when the last message is unknown", func(t *testing.T) {
		node := agent.NewPeer("node1")
		stale := func(m *agent.Message) *agent.Message {
			past := time.Now().Add(-time.Hour)
			m.Id, m.Ts = uuid.Must(uuid.NewV7AtTime(past)).String(), past.UnixNano()
			return m
		}

		id := []byte("ours")
		deploymentID := new(atomic.Pointer[[]byte])
		deploymentID.Store(&id)

		buf := make(chan *agent.Message, 3)
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: id}, Options: &agent.DeployOptions{}})
		// a completion from before the client started would finish the monitor if it were replayed.
		buf <- agent.NewLogHistoryFromMessages(
			agent.NewPeer("local"),
			stale(agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: id}, Options: &agent.DeployOptions{}})),
		)

		ctx, timeout := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer timeout()
		ctx = contextx.NewWaitGroup(ctx)
		ctx, failed := context.WithCancelCause(ctx)
		Deploy(ctx, failed, nil, buf, agent.NewPeer("local"), OptionDeployment(deploymentID))
		require.ErrorIs(t, context.Cause(ctx), context.DeadlineExceeded)
	})

	t.Run("should resume history replay from the last message before a heartbeat", func(t *testing.T) {
		local := agent.NewPeer("local")
		node := agent.NewPeer("node1")
		begin := agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}})
		completed := agent.PeersCompletedEvent(node, 1)
		done := agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}})

		buf := make(chan *agent.Message, 4)
		buf <- begin
		buf <- completed
		// heartbeats are delivered live but never recorded in the history.
		buf <- agent.NewDeployHeartbeat(node)
		buf <- agent.NewLogHistoryFromMessages(local, begin, completed, done)

		ctx, timeout := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer timeout()
		ctx = contextx.NewWaitGroup(ctx)
		ctx, failed := context.WithCancelCause(ctx)
		Deploy(ctx, failed, nil, buf, agent.NewPeer("local"))
		require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))
	})

	t.Run("should ignore the history until the deployment id is known", func(t *testing.T) {
		node := agent.NewPeer("node1")
		buf := make(chan *agent.Message, 2)
		buf <- agent.NewConnectionLog(agent.NewPeer("local"), agent.ConnectionEvent_Connected, "connected")
		buf <- agent.NewLogHistoryFromMessages(
			agent.NewPeer("local"),
			agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
		)

		ctx, timeout := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer timeout()
		ctx = contextx.NewWaitGroup(ctx)
		ctx, failed := context.WithCancelCause(ctx)
		// the id is only stored once the archive has been uploaded.
		Deploy(ctx, failed, nil, buf, node, OptionDeployment(new(atomic.Pointer[[]byte])))
		require.ErrorIs(t, context.Cause(ctx), context.DeadlineExceeded)
	})

	t.Run("should fill the upload progress bar with # by default", func(t *testing.T) {
		node := agent.NewPeer("node1")
		buf := make(chan *agent.Message, 3)
		buf <- agent.NewUploadProgress(node, 1<<20, 2<<20)
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}})
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}})

		output := bytes.NewBuffer(nil)
		ctx, timeout := context.WithTimeout(context.Background(), time.Second)
		defer timeout()
		ctx = contextx.NewWaitGroup(ctx)
		ctx, failed := context.WithCancelCause(ctx)
		Deploy(ctx, failed, nil, buf, node, OptionLogger(log.New(output, "", 0)))
		require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))

		require.Contains(t, output.String(), "archive upload ["+strings.Repeat("#", 15)+strings.Repeat("-", 15)+"]  50%")
	})

	t.Run("should keep the completed upload progress", func(t *testing.T) {
		node := agent.NewPeer("node1")
		archive := &agent.Archive{DeploymentID: []byte("ours")}
		buf := make(chan *agent.Message, 4)
		buf <- agent.NewUploadProgress(node, 1<<20, 2<<20)
		buf <- agent.NewUploadProgress(node, 2<<20, 2<<20)
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: archive, Options: &agent.DeployOptions{}})
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: archive, Options: &agent.DeployOptions{}})

		output := bytes.NewBuffer(nil)
		ctx, timeout := context.WithTimeout(context.Background(), time.Second)
		defer timeout()
		ctx = contextx.NewWaitGroup(ctx)
		ctx, failed := context.WithCancelCause(ctx)
		Deploy(ctx, failed, nil, buf, agent.NewPeer("local"), OptionLogger(log.New(output, "", 0)))
		require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))

		// in progress updates are replaced by the next update, the completed one remains.
		require.Regexp(t, ` 50% \(1\.0 MiB / 2\.0 MiB\) \([0-9a-z.]+\)\r`, output.String())
		require.Regexp(t, `100% \(2\.0 MiB / 2\.0 MiB\) \([0-9a-z.]+\)\n`, output.String())
		require.NotRegexp(t, `100% \(2\.0 MiB / 2\.0 MiB\) \([0-9a-z.]+\)\r`, output.String())
	})

	t.Run("should fill the upload progress bar with sparkles", func(t *testing.T) {
		node := agent.NewPeer("node1")
		archive := &agent.Archive{DeploymentID: []byte("ours")}
		buf := make(chan *agent.Message, 4)
		buf <- agent.NewUploadProgress(node, 1<<20, 2<<20)
		buf <- agent.NewUploadProgress(node, 2<<20, 2<<20)
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: archive, Options: &agent.DeployOptions{}})
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: archive, Options: &agent.DeployOptions{}})

		output := bytes.NewBuffer(nil)
		ctx, timeout := context.WithTimeout(context.Background(), time.Second)
		defer timeout()
		ctx = contextx.NewWaitGroup(ctx)
		ctx, failed := context.WithCancelCause(ctx)
		Deploy(ctx, failed, nil, buf, agent.NewPeer("local"), OptionLogger(log.New(output, "", 0)), OptionIconography('✨'))
		require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))

		// half way is 15 of the 30 cells filled, each empty cell is two dashes to match the width of the sparkle.
		require.Contains(t, output.String(), "archive upload ["+strings.Repeat("✨", 15)+strings.Repeat("--", 15)+"]  50%")
		require.Contains(t, output.String(), "archive upload ["+strings.Repeat("✨", 30)+"] 100%")
	})

	t.Run("should only print the completed upload progress once", func(t *testing.T) {
		node := agent.NewPeer("node1")
		archive := &agent.Archive{DeploymentID: []byte("ours")}
		buf := make(chan *agent.Message, 8)
		buf <- agent.NewUploadProgress(node, 1<<20, 2<<20)
		buf <- agent.NewUploadProgress(node, 2<<20, 2<<20)
		buf <- agent.NewUploadProgress(node, 2<<20, 2<<20)
		buf <- agent.NewUploadProgress(node, 2<<20, 2<<20)
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: archive, Options: &agent.DeployOptions{}})
		buf <- agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: archive, Options: &agent.DeployOptions{}})

		output := bytes.NewBuffer(nil)
		ctx, timeout := context.WithTimeout(context.Background(), time.Second)
		defer timeout()
		ctx = contextx.NewWaitGroup(ctx)
		ctx, failed := context.WithCancelCause(ctx)
		Deploy(ctx, failed, nil, buf, agent.NewPeer("local"), OptionLogger(log.New(output, "", 0)))
		require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))

		require.Equal(t, 1, strings.Count(output.String(), "] 100%"))
	})

	t.Run("should resume history replay across consecutive reconnects", func(t *testing.T) {
		local := agent.NewPeer("local")
		node := agent.NewPeer("node1")
		begin := agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}})
		done := agent.NewDeployCommand(node, &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}})

		buf := make(chan *agent.Message, 3)
		buf <- begin
		// reconnect with nothing new since begin.
		buf <- agent.NewLogHistoryFromMessages(local, begin)
		// reconnect after the deploy completed while disconnected.
		buf <- agent.NewLogHistoryFromMessages(local, begin, done)

		ctx, timeout := context.WithTimeout(context.Background(), time.Second)
		defer timeout()
		ctx = contextx.NewWaitGroup(ctx)
		ctx, failed := context.WithCancelCause(ctx)
		Deploy(ctx, failed, nil, buf, agent.NewPeer("local"))
		require.NoError(t, errorsx.Ignore(context.Cause(ctx), context.Canceled))
	})
}

func TestMain(m *testing.M) {
	go debugx.DumpOnSignal(context.Background(), syscall.SIGUSR2)
	log.SetOutput(io.Discard)
	code := m.Run()
	testingx.Cleanup()
	os.Exit(code)
}
