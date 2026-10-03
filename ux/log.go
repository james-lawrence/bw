package ux

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/stringsx"
	"github.com/logrusorgru/aurora"
)

// Option ...
type Option func(*cState)

// OptionFailureDisplay ...
func OptionFailureDisplay(fd failureDisplay) Option {
	return func(c *cState) {
		c.FailureDisplay = fd
	}
}

func OptionHeartbeat(d time.Duration) Option {
	return func(cs *cState) {
		cs.heartbeat = 3 * d
	}
}

// OptionDeployment ignore deploy commands for other deployments.
// the id is provided once known (e.g. after the archive is uploaded).
func OptionDeployment(id *atomic.Pointer[[]byte]) Option {
	return func(cs *cState) {
		cs.deploymentID = id
		cs.deploying = new(bool)
	}
}

func OptionDebug(b bool) Option {
	return func(cs *cState) {
		cs.debug = b
	}
}

// Deploy monitor a deploy.
func Deploy(ctx context.Context, failed context.CancelCauseFunc, cached *dialers.Cached, events chan *agent.Message, options ...Option) {
	var (
		s = deploying{
			cState: cState{
				cached:         cached,
				heartbeat:      time.Minute,
				connection:     &agent.ConnectionEvent{},
				FailureDisplay: FailureDisplayNoop{},
				au:             aurora.NewAurora(true),
				Logger:         log.New(os.Stderr, "[CLIENT] ", 0),
				failed:         failed,
			}.merge(options...),
		}
	)

	defer failed(nil)
	s.run(ctx, events, s)
}

// Logging based ux
func Logging(ctx context.Context, failed context.CancelCauseFunc, cached *dialers.Cached, events chan *agent.Message, options ...Option) {
	var (
		s = tail{
			cState: cState{
				cached:         cached,
				heartbeat:      time.Minute,
				connection:     &agent.ConnectionEvent{},
				FailureDisplay: FailureDisplayNoop{},
				au:             aurora.NewAurora(true),
				Logger:         log.New(os.Stderr, "[CLIENT] ", 0),
				failed:         failed,
			}.merge(options...),
		}
	)

	defer failed(nil)
	defer log.Println("run completed")
	s.run(ctx, events, s)
}

func (t cState) run(ctx context.Context, events chan *agent.Message, s consumer) {
	var (
		last *agent.Message
	)

	for {
		select {
		case <-time.After(t.heartbeat):
			t.Logger.Println(
				t.au.Yellow(fmt.Sprintf("no message has been received within the time limit %s", t.heartbeat)),
			)
			t.cached.Close()
		case m := <-events:
			switch local := m.Event.(type) {
			case *agent.Message_History:
				replayable := slice(last, local.History.Messages...)
				s = consume(s, replayable...)
				last = latest(last, replayable...)
			default:
				s = consume(s, m)
				last = latest(last, m)
			}

			if s == nil {
				// we're done
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// latest returns the most recent message a history replay can resume from.
// log events are skipped, as are history messages since they're generated
// by the client and never appear in the history itself.
func latest(last *agent.Message, messages ...*agent.Message) *agent.Message {
	for _, m := range messages {
		switch m.Type {
		case agent.Message_LogEvent, agent.Message_LogHistoryEvent:
		default:
			last = m
		}
	}

	return last
}

func slice(last *agent.Message, messages ...*agent.Message) []*agent.Message {
	if last == nil {
		return []*agent.Message{}
	}

	consumable := false
	for idx, m := range messages {
		if consumable {
			return messages[idx:]
		}

		consumable = m.Id == last.Id
	}

	return []*agent.Message{}
}

func consume(c consumer, messages ...*agent.Message) consumer {
	for _, m := range messages {
		// log.Println("consuming", messageDebug(m))
		if c = c.Consume(m); c == nil {
			// we're done
			return nil
		}
	}
	return c
}

type cState struct {
	cached         *dialers.Cached
	connection     *agent.ConnectionEvent
	deploymentID   *atomic.Pointer[[]byte]
	deploying      *bool
	FailureDisplay failureDisplay
	Logger         *log.Logger
	au             aurora.Aurora
	heartbeat      time.Duration
	debug          bool
	failed         context.CancelCauseFunc
}

func (t cState) merge(options ...Option) cState {
	dup := t

	for _, opt := range options {
		opt(&dup)
	}

	return dup
}

// relevant reports if the message applies to the monitored deployment.
// messages that carry an archive must match the deployment's id. cancel and restart
// commands carry no archive and apply to whichever deployment is running, so they're
// only relevant once the monitored deployment has begun.
// when no deployment is being monitored every message is relevant.
func (t cState) relevant(m *agent.Message) bool {
	var (
		archive *agent.Archive
	)

	if t.deploymentID == nil {
		return true
	}

	switch m.Type {
	case agent.Message_DeployCommandEvent:
		archive = m.GetDeployCommand().Archive
	case agent.Message_DeployEvent:
		archive = m.GetDeploy().Archive
	default:
		return true
	}

	if archive == nil {
		return *t.deploying
	}

	id := t.deploymentID.Load()
	matched := id != nil && bytes.Equal(*id, archive.DeploymentID)

	if m.Type == agent.Message_DeployCommandEvent && m.GetDeployCommand().Command == agent.DeployCommand_Begin {
		*t.deploying = matched
	}

	return matched
}

func (t cState) print(m *agent.Message) {
	switch evt := m.Event.(type) {
	case *agent.Message_Int:
		switch m.Type {
		case agent.Message_PeersCompletedEvent:
			// Do nothing.
		case agent.Message_PeersFoundEvent:
			t.Logger.Printf(
				"%s - INFO - located %d peers\n",
				messagePrefix(m),
				m.GetInt(),
			)
		default:
			t.Logger.Printf("%s - %s\n", messagePrefix(m), m.Type)
		}
	case *agent.Message_Log:
		t.Logger.Printf(
			"%s - INFO - %s\n",
			messagePrefix(m),
			evt.Log.Log,
		)
	case *agent.Message_DeployCommand:
		t.printDeployCommand(m)
	case *agent.Message_Deploy:
		t.Logger.Printf(
			"%s - %s %s\n",
			messagePrefix(m),
			evt.Deploy.Stage,
			bw.RandomID(evt.Deploy.Archive.DeploymentID),
		)
	case *agent.Message_Membership:
		t.Logger.Println(
			t.au.Yellow(
				fmt.Sprintf("%s - INFO - %s",
					messagePrefix(m),
					evt.Membership,
				),
			),
		)
	case *agent.Message_History:
	case *agent.Message_Connection:
		if t.connection.State == evt.Connection.State {
			return
		}

		t.Logger.Printf(
			"%s - INFO - %s -> %s %s\n",
			messagePrefix(m),
			t.connection.State,
			evt.Connection.State,
			evt.Connection.Description,
		)

		t.connection.State = evt.Connection.State
	case *agent.Message_Heartbeat:
		if t.debug {
			t.Logger.Printf("%s - %s\n", messagePrefix(m), m.Type)
		}
	default:
		t.Logger.Printf("%s - %s\n", messagePrefix(m), m.Type)
	}
}

func (t cState) printDeployCommand(m *agent.Message) {
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
		did := ""
		if d.Archive != nil {
			did = bw.RandomID(d.Archive.DeploymentID).String()
		}
		t.Logger.Println(
			t.au.Red(fmt.Sprintf("%s - INFO - deployment failed - %s", messagePrefix(m), did)),
		)
	case agent.DeployCommand_Cancel:
		t.Logger.Println(
			t.au.Red(fmt.Sprintf("%s - INFO - deployment cancelled by %s", messagePrefix(m), stringsx.DefaultIfBlank(d.Initiator, "agent"))),
		)
	case agent.DeployCommand_Restart:
		t.Logger.Println(
			t.au.Yellow(fmt.Sprintf("%s - INFO - deployment restarted by %s", messagePrefix(m), stringsx.DefaultIfBlank(d.Initiator, "agent"))),
		)
	default:
		log.Println("unexpected command", messagePrefix(m), spew.Sdump(m))
	}
}

type consumer interface {
	Consume(*agent.Message) consumer
}

type tail struct {
	cState
}

func (t tail) Consume(m *agent.Message) consumer {
	t.cState.print(m)
	return t
}

type deploying struct {
	cState
}

func (t deploying) Consume(m *agent.Message) consumer {
	t.cState.print(m)

	if !t.cState.relevant(m) {
		return t
	}

	switch m.Type {
	case agent.Message_DeployCommandEvent:
		switch m.GetDeployCommand().Command {
		case agent.DeployCommand_Restart:
			return restart(t)
		case
			agent.DeployCommand_Done,
			agent.DeployCommand_Cancel,
			agent.DeployCommand_Failed:
			return nil
		default:
			// log.Println("unhandled deploy command", m.GetDeployCommand().Command)
		}
	case agent.Message_DeployEvent:
		d := m.GetDeploy()
		switch d.Stage {
		case agent.Deploy_Failed:
			digest := md5.Sum([]byte(d.Error))
			return failure{
				cState: t.cState,
				failures: map[string]*agent.Message{
					hex.EncodeToString(digest[:]): m,
				},
			}
		default:
		}
	}

	// await next message by default
	return t
}
