package ux

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/stringsx"
	"github.com/logrusorgru/aurora"
)

type dialer interface {
	Dial(p agent.Peer) (agent.Client, error)
}

// Option ...
type Option func(*cState)

// OptionFailureDisplay ...
func OptionFailureDisplay(fd failureDisplay) Option {
	return func(c *cState) {
		c.FailureDisplay = fd
	}
}

// Deploy monitor a deploy.
func Deploy(ctx context.Context, wg *sync.WaitGroup, events chan agent.Message, options ...Option) {
	defer wg.Done()

	var (
		s consumer = deploying{
			cState: cState{
				FailureDisplay: FailureDisplayNoop{},
				au:             aurora.NewAurora(true),
				Logger:         log.New(os.Stderr, "[CLIENT] ", 0),
			}.merge(options...),
		}
	)

	run(ctx, events, s)
}

// Logging based ux
func Logging(ctx context.Context, wg *sync.WaitGroup, events chan agent.Message, options ...Option) {
	defer wg.Done()

	var (
		s consumer = tail{
			cState: cState{
				FailureDisplay: FailureDisplayNoop{},
				au:             aurora.NewAurora(true),
				Logger:         log.New(os.Stderr, "[CLIENT] ", 0),
			}.merge(options...),
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
	FailureDisplay failureDisplay
	Logger         *log.Logger
	au             aurora.Aurora
}

func (t cState) merge(options ...Option) cState {
	dup := t

	for _, opt := range options {
		opt(&dup)
	}

	return dup
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
	d := m.GetDeployCommand()
	switch d.Command {
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
	case agent.DeployCommand_Cancel:
		t.Logger.Println(
			t.au.Red(fmt.Sprintf("%s - INFO - deployment cancelled by %s", messagePrefix(m), stringsx.DefaultIfBlank(d.Initiator, "agent"))),
		)
	default:
		log.Println("unexpected command", messagePrefix(m), spew.Sdump(m))
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
		case agent.DeployCommand_Restart:
			return restart{cState: t.cState}
		case
			agent.DeployCommand_Done,
			agent.DeployCommand_Cancel:
			return nil
		}
	case agent.Message_DeployEvent:
		d := m.GetDeploy()
		switch d.Stage {
		case agent.Deploy_Failed:
			digest := md5.Sum([]byte(d.Error))
			return failure{
				cState: t.cState,
				failures: map[string]agent.Message{
					hex.EncodeToString(digest[:]): m,
				},
			}
		}
	}

	// await next message by default
	return t
}
