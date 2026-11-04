package ux

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/bytesx"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/langx"
	"github.com/james-lawrence/bw/internal/slogx"
	"github.com/james-lawrence/bw/internal/stringsx"
	"github.com/logrusorgru/aurora"
	"github.com/mattn/go-runewidth"
)

const (
	uploading = "\r"
	uploaded  = "\n"
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

// OptionLogger sets the logger used to display messages.
func OptionLogger(l *log.Logger) Option {
	return func(cs *cState) {
		cs.Logger = l
	}
}

// OptionStarted sets when the client started, history older than this is never replayed for
// a deployment that was already being followed. defaults to the time the monitor was created.
func OptionStarted(t time.Time) Option {
	return func(cs *cState) {
		cs.started = t
	}
}

// OptionIconography sets the rune used to fill the upload progress bar.
func OptionIconography(r rune) Option {
	return func(cs *cState) {
		cs.iconography = r
	}
}

func OptionDebug(b bool) Option {
	return func(cs *cState) {
		cs.debug = b
	}
}

// Deploy monitor a deploy.
func Deploy(ctx context.Context, failed context.CancelCauseFunc, cached *dialers.Cached, events chan *agent.Message, local *agent.Peer, options ...Option) {
	var (
		s = deploying{
			cState: cState{
				local:           local,
				iconography:     envx.Rune('#', bw.EnvUploadSparkles),
				cached:          cached,
				heartbeat:       time.Minute,
				connection:      &agent.ConnectionEvent{},
				FailureDisplay:  FailureDisplayNoop{},
				au:              aurora.NewAurora(true),
				Logger:          log.New(os.Stderr, "[CLIENT] ", 0),
				failed:          failed,
				started:         time.Now(),
				uploadcompleted: new(uploading),
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
				iconography:    envx.Rune('#', bw.EnvUploadSparkles),
				cached:         cached,
				heartbeat:      time.Minute,
				connection:     &agent.ConnectionEvent{},
				FailureDisplay: FailureDisplayNoop{},
				au:             aurora.NewAurora(true),
				Logger:         log.New(os.Stderr, "[CLIENT] ", 0),
				failed:         failed,
				started:        time.Now(),
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
			s = consume(s, agent.LogWarning(t.local, t.au.Yellow(fmt.Sprintf("no message has been received within the time limit %s", t.heartbeat)).String()))
			t.cached.Close()
		case m := <-events:
			switch local := m.Event.(type) {
			case *agent.Message_History:
				replayable := t.slice(last, local.History.Messages...)
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
// by the client and heartbeats since they're never recorded in the history.
func latest(last *agent.Message, messages ...*agent.Message) *agent.Message {
	for _, m := range messages {
		switch m.Type {
		case agent.Message_LogEvent, agent.Message_LogHistoryEvent, agent.Message_DeployHeartbeat, agent.Message_UploadEvent:
		default:
			last = m
		}
	}

	return last
}

// slice returns the messages after last. when last isn't in the history and a deployment
// is being monitored the history since that deployment's begin is returned instead.
// history from other deployments, including everything before the monitored deployment's
// id is known (e.g. while the archive is uploading), is never replayed.
func (t cState) slice(last *agent.Message, messages ...*agent.Message) []*agent.Message {
	if last != nil {
		for idx, m := range messages {
			if m.Id == last.Id {
				return messages[idx+1:]
			}
		}
	}

	if t.deploymentID == nil {
		return []*agent.Message{}
	}

	id := t.deploymentID.Load()
	if id == nil {
		return []*agent.Message{}
	}

	for idx, m := range slices.Backward(messages) {
		if m.Type != agent.Message_DeployCommandEvent {
			continue
		}

		if d := m.GetDeployCommand(); d.Command == agent.DeployCommand_Begin && d.Archive != nil && bytes.Equal(d.Archive.DeploymentID, *id) {
			return messages[idx:]
		}
	}

	if last != nil {
		// the deployment was already being followed but its begin has left the history. anything from
		// before this client started belongs to a previous deployment.
		return slices.DeleteFunc(slices.Clone(messages), func(m *agent.Message) bool {
			return agent.MessageTime(m).Before(t.started)
		})
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
	iconography     rune
	local           *agent.Peer
	cached          *dialers.Cached
	connection      *agent.ConnectionEvent
	deploymentID    *atomic.Pointer[[]byte]
	deploying       *bool
	FailureDisplay  failureDisplay
	Logger          *log.Logger
	au              aurora.Aurora
	heartbeat       time.Duration
	debug           bool
	failed          context.CancelCauseFunc
	uploadcompleted *string
	started         time.Time
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
	case *agent.Message_Upload:
		t.printUploadProgress(m, evt.Upload)
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
		slogx.With(t.Logger).If(t.debug).Printf("%s - %s\n", messagePrefix(m), m.Type)
	default:
		t.Logger.Printf("%s - %s\n", messagePrefix(m), m.Type)
	}
}

// printUploadProgress renders a single line progress bar. the trailing carriage return leaves the cursor
// at the start of the line so the next update, or the next message, replaces the bar. the erase line clears
// anything left over from a previous write. the completed bar ends with a newline so it remains.
func (t cState) printUploadProgress(m *agent.Message, u *agent.UploadProgress) {
	const width = 30

	pct := min(100, u.Bytes*100/max(1, u.Total))

	if *t.uploadcompleted == uploaded {
		return
	} else if pct >= 100 {
		*t.uploadcompleted = uploaded
	} else {
		*t.uploadcompleted = uploading
	}

	filled := int(pct) * width / 100
	// the empty cells match the display width of the iconography (e.g. emoji are two columns wide),
	// so the bar doesn't change width as it fills.
	cell := max(1, runewidth.RuneWidth(t.iconography))
	fmt.Fprintf(
		t.Logger.Writer(),
		"\x1b[2K%s%s - INFO - archive upload [%s%s] %3d%% (%s / %s) (%s)%s",
		t.Logger.Prefix(),
		messagePrefix(m),
		strings.Repeat(string(t.iconography), filled),
		strings.Repeat("-", (width-filled)*cell),
		pct,
		bytesx.Human(u.Bytes),
		bytesx.Human(u.Total),
		time.Since(t.started).Round(time.Second),
		langx.Autoderef(t.uploadcompleted),
	)
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
