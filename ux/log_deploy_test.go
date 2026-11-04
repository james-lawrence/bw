package ux_test

import (
	"bytes"
	"context"
	"log"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/contextx"
	"github.com/james-lawrence/bw/internal/errorsx"
	. "github.com/james-lawrence/bw/ux"
)

var _ = Describe("Log Deploy", func() {
	DescribeTable("should process every message",
		func(failure error, messages ...*agent.Message) {
			buf := make(chan *agent.Message, len(messages))
			for _, m := range messages {
				buf <- m
			}
			ctx := contextx.NewWaitGroup(context.Background())
			ctx, failed := context.WithCancelCause(ctx)
			Deploy(ctx, failed, nil, buf, agent.NewPeer("local"))
			Expect(len(buf)).To(Equal(0))
			if failure != nil {
				Expect(context.Cause(ctx)).To(MatchError(failure))
			} else {
				Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())
			}
		},
		Entry(
			"successful deploy",
			error(nil),
			agent.LogEvent(agent.NewPeer("node1"), "hello world"),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agent.LogEvent(agent.NewPeer("node1"), "info message"),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
		),
		Entry(
			"failed deploy",
			errorsx.String("deploy failed"),
			agent.LogEvent(agent.NewPeer("node1"), "hello world"),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agent.LogEvent(agent.NewPeer("node1"), "info message"),
			agent.DeployEvent(agent.NewPeer("node1"), &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}, Error: "boom"}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
		),
		Entry(
			"automatic restart deploy",
			error(nil),
			agent.LogEvent(agent.NewPeer("node1"), "hello world"),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agent.LogEvent(agent.NewPeer("node1"), "info message"),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Restart, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agent.LogEvent(agent.NewPeer("node1"), "info message"),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Cancel, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agent.LogEvent(agent.NewPeer("node1"), "info message"),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
		),
		Entry(
			"automatic restart after a node failure",
			error(nil),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agent.DeployEvent(agent.NewPeer("node1"), &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}, Error: "boom"}),
			agent.NewDeployCommand(agent.NewPeer("node1"), agent.DeployCommandRestart()),
			agent.NewDeployCommand(agent.NewPeer("node1"), agent.DeployCommandCancel("")),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
		),
	)

	DescribeTable("should only finish on commands for the monitored deployment",
		func(failure error, messages ...*agent.Message) {
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
			Expect(len(buf)).To(Equal(0))
			if failure != nil {
				Expect(context.Cause(ctx)).To(MatchError(failure))
			} else {
				Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())
			}
		},
		Entry(
			"previous deploy completed",
			error(nil),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
		),
		Entry(
			"previous deploy cancelled",
			error(nil),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), agent.DeployCommandCancel("someone")),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
		),
		Entry(
			"previous deploy failed",
			error(nil),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			agent.DeployEvent(agent.NewPeer("node1"), &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}, Error: "boom"}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
		),
		Entry(
			"cancelled",
			error(nil),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			agent.NewDeployCommand(agent.NewPeer("node1"), agent.DeployCommandCancel("someone")),
		),
		Entry(
			"failed",
			errorsx.String("deploy failed"),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			agent.DeployEvent(agent.NewPeer("node1"), &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}, Error: "boom"}),
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
		),
	)

	DescribeTable("should replay history when the last message is unknown",
		func(monitored bool, finished bool, messages ...*agent.Message) {
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
				Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())
			} else {
				Expect(context.Cause(ctx)).To(MatchError(context.DeadlineExceeded))
			}
		},
		Entry(
			"first connect after the deploy completed",
			true, true,
			agent.NewLogHistoryFromMessages(
				agent.NewPeer("local"),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			),
		),
		Entry(
			"last message no longer in the history",
			true, true,
			agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			agent.NewLogHistoryFromMessages(
				agent.NewPeer("local"),
				agent.PeersCompletedEvent(agent.NewPeer("node1"), 1),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("ours")}, Options: &agent.DeployOptions{}}),
			),
		),
		Entry(
			"history of only a previous deployment is ignored",
			true, false,
			agent.NewLogHistoryFromMessages(
				agent.NewPeer("local"),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			),
		),
		Entry(
			"first connect without a monitored deployment ignores the history",
			false, false,
			agent.NewLogHistoryFromMessages(
				agent.NewPeer("local"),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
				agent.NewDeployCommand(agent.NewPeer("node1"), &agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{DeploymentID: []byte("previous")}, Options: &agent.DeployOptions{}}),
			),
		),
	)

	It("should not replay history from before the client started when the last message is unknown", func() {
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
		Expect(context.Cause(ctx)).To(MatchError(context.DeadlineExceeded))
	})

	It("should resume history replay from the last message before a heartbeat", func() {
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
		Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())
	})

	It("should ignore the history until the deployment id is known", func() {
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
		Expect(context.Cause(ctx)).To(MatchError(context.DeadlineExceeded))
	})

	It("should fill the upload progress bar with # by default", func() {
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
		Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())

		Expect(output.String()).To(ContainSubstring("archive upload [" + strings.Repeat("#", 15) + strings.Repeat("-", 15) + "]  50%"))
	})

	It("should keep the completed upload progress", func() {
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
		Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())

		// in progress updates are replaced by the next update, the completed one remains.
		Expect(output.String()).To(MatchRegexp(` 50% \(1\.0 MiB / 2\.0 MiB\) \([0-9a-z.]+\)\r`))
		Expect(output.String()).To(MatchRegexp(`100% \(2\.0 MiB / 2\.0 MiB\) \([0-9a-z.]+\)\n`))
		Expect(output.String()).ToNot(MatchRegexp(`100% \(2\.0 MiB / 2\.0 MiB\) \([0-9a-z.]+\)\r`))
	})

	It("should fill the upload progress bar with sparkles", func() {
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
		Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())

		// half way is 15 of the 30 cells filled, each empty cell is two dashes to match the width of the sparkle.
		Expect(output.String()).To(ContainSubstring("archive upload [" + strings.Repeat("✨", 15) + strings.Repeat("--", 15) + "]  50%"))
		Expect(output.String()).To(ContainSubstring("archive upload [" + strings.Repeat("✨", 30) + "] 100%"))
	})

	It("should only print the completed upload progress once", func() {
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
		Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())

		Expect(strings.Count(output.String(), "] 100%")).To(Equal(1))
	})

	It("should resume history replay across consecutive reconnects", func() {
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
		Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())
	})
})
