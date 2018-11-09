package ux

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/x/errorsx"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
)

func Deploy(ctx context.Context, wg *sync.WaitGroup, d agent.Dialer, events chan agent.Message) {
	defer wg.Done()
	var (
		s consumer = deploying{
			cState: cState{
				au:     aurora.NewAurora(true),
				Dialer: d,
				Logger: log.New(os.Stderr, "[CLIENT] ", 0),
			},
		}
	)

	run(ctx, events, s)
}

// Logging based ux
func Logging(ctx context.Context, wg *sync.WaitGroup, d agent.Dialer, events chan agent.Message) {
	defer wg.Done()
	var (
		s consumer = tail{
			cState: cState{
				au:     aurora.NewAurora(true),
				Dialer: d,
				Logger: log.New(os.Stderr, "[CLIENT] ", 0),
			},
		}
	)

	run(ctx, events, s)
}

func run(ctx context.Context, events chan agent.Message, s consumer) {
	for {
		select {
		case m := <-events:
			if s = s.Consume(m); s == nil {
				// we're done
				return
			}
		case _ = <-ctx.Done():
			return
		}
	}
}

type cState struct {
	Dialer agent.Dialer
	Logger *log.Logger
	au     aurora.Aurora
}

func (t cState) print(m agent.Message) {
	switch m.Type {
	case agent.Message_PeerEvent:
	case agent.Message_DeployCommandEvent:
		t.printDeployCommand(m)
	case agent.Message_PeersCompletedEvent:
		// Do nothing.
	case agent.Message_PeersFoundEvent:
		t.Logger.Printf(
			"%s - INFO - located %d peers\n",
			messagePrefix(m),
			m.GetInt(),
		)
	case agent.Message_DeployEvent:
		d := m.GetDeploy()
		t.Logger.Printf(
			"%s - %s %s\n",
			messagePrefix(m),
			d.Stage,
			bw.RandomID(d.Archive.DeploymentID),
		)
	case agent.Message_LogEvent:
		d := m.GetLog()
		t.Logger.Printf(
			"%s - INFO - %s\n",
			messagePrefix(m),
			d.Log,
		)
	default:
		t.Logger.Printf("%s - %s\n", messagePrefix(m), m.Type)
	}
}

func (t cState) printDeployCommand(m agent.Message) {
	switch m.GetDeployCommand().Command {
	case agent.DeployCommand_Begin:
		t.Logger.Println(
			t.au.Green(fmt.Sprintf("%s - INFO - deployment initiated", messagePrefix(m))),
		)
	case agent.DeployCommand_Done:
		t.Logger.Println(
			t.au.Green(fmt.Sprintf("%s - INFO - deployment completed", messagePrefix(m))),
		)
	case agent.DeployCommand_Failed:
		t.Logger.Println(
			t.au.Red(fmt.Sprintf("%s - INFO - deployment failed", messagePrefix(m))),
		)
	}
}

type consumer interface {
	Consume(agent.Message) consumer
}

type tail struct {
	cState
}

func (t tail) Consume(m agent.Message) consumer {
	t.cState.print(m)
	return t
}

type deploying struct {
	cState
}

func (t deploying) Consume(m agent.Message) consumer {
	t.cState.print(m)
	switch m.Type {
	case agent.Message_DeployCommandEvent:
		switch m.GetDeployCommand().Command {
		case agent.DeployCommand_Done:
			return nil
		}
	case agent.Message_DeployEvent:
		d := m.GetDeploy()
		switch d.Stage {
		case agent.Deploy_Failed:
			return failure{failures: []agent.Message{m}, cState: t.cState}
		}
	}

	// await next message by default
	return t
}

// gathers up failures
type failure struct {
	failures []agent.Message
	cState
}

func (t failure) Consume(m agent.Message) consumer {
	switch m.Type {
	case agent.Message_DeployCommandEvent:
		t.logs()
		t.cState.printDeployCommand(m)
		return nil // done.
	case agent.Message_DeployEvent:
		d := m.GetDeploy()
		switch d.Stage {
		case agent.Deploy_Failed:
			return failure{failures: append(t.failures, m), cState: t.cState}
		}
	}

	return t
}

func (t failure) logs() {
	for i, m := range t.failures {
		var (
			err error
			c   agent.Client
		)

		if m.Peer == nil {
			log.Println("unexpected nil peer skipping", spew.Sdump(m))
			continue
		}

		if c, err = t.cState.Dialer.Dial(*m.Peer); err != nil {
			log.Println(errors.Wrapf(err, "failed to dial peer: %s", spew.Sdump(m.Peer)))
		}

		if i > 0 {
			os.Stderr.WriteString("\n\n")
		}

		t.cState.Logger.Println(t.cState.au.Brown(fmt.Sprint("BEGIN LOGS:", messagePrefix(m))))
		logx.MaybeLog(
			errorsx.Compact(
				agentutil.PrintLogs(m.GetDeploy().Archive.DeploymentID, os.Stderr)(c),
				c.Close(),
			),
		)
		t.cState.Logger.Println(t.cState.au.Brown(fmt.Sprint("CEASE LOGS:", messagePrefix(m))))
	}
}
