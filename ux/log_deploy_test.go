package ux_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

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
			Deploy(ctx, failed, nil, buf)
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
	)

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
		Deploy(ctx, failed, nil, buf)
		Expect(errorsx.Ignore(context.Cause(ctx), context.Canceled)).To(Succeed())
	})
})
