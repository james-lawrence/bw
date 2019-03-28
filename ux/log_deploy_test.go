package ux_test

import (
	"context"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	. "github.com/james-lawrence/bw/ux"
)

type errDialer struct {
	cause error
}

func (t errDialer) Dial(agent.Peer) (agent.Client, error) {
	return nil, t.cause
}

var _ = Describe("Log Deploy", func() {
	DescribeTable("should process every message",
		func(messages ...agent.Message) {
			buf := make(chan agent.Message, len(messages))
			for _, m := range messages {
				buf <- m
			}
			wg := &sync.WaitGroup{}
			wg.Add(1)
			Deploy(context.Background(), wg, buf)
			Expect(len(buf)).To(Equal(0))
		},
		Entry(
			"successful deploy",
			agentutil.LogEvent(agent.NewPeer("node1"), "hello world"),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agentutil.LogEvent(agent.NewPeer("node1"), "info message"),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
		),
		Entry(
			"failed deploy",
			agentutil.LogEvent(agent.NewPeer("node1"), "hello world"),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agentutil.LogEvent(agent.NewPeer("node1"), "info message"),
			agentutil.DeployEvent(agent.NewPeer("node1"), agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}, Error: "boom"}),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
		),
		Entry(
			"automatic restart deploy",
			agentutil.LogEvent(agent.NewPeer("node1"), "hello world"),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agentutil.LogEvent(agent.NewPeer("node1"), "info message"),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Restart, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agentutil.LogEvent(agent.NewPeer("node1"), "info message"),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Cancel, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
			agentutil.LogEvent(agent.NewPeer("node1"), "info message"),
			agentutil.DeployCommand(agent.NewPeer("node1"), agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}}),
		),
	)
})
