package agent

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// PeersFoundEvent ...
func PeersFoundEvent(p *Peer, n int64) *Message {
	return integerEvent(p, Message_PeersFoundEvent, n)
}

// PeersCompletedEvent ...
func PeersCompletedEvent(p *Peer, n int64) *Message {
	return integerEvent(p, Message_PeersCompletedEvent, n)
}

// LogEvent create a log event message.
func LogEvent(p *Peer, s string) *Message {
	return &Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: Message_LogEvent,
		Peer: p,
		Ts:   time.Now().Unix(),
		Event: &Message_Log{
			Log: &Log{
				Log: s,
			},
		},
	}
}

// NewTLSCertificates ...
func NewTLSCertificates(auth, cert []byte) TLSCertificates {
	digest := md5.Sum(cert)
	return TLSCertificates{
		Fingerprint: hex.EncodeToString(digest[:]),
		Authority:   auth,
		Certificate: cert,
	}
}

// NewWALPreamble preamble message for the WAL.
func NewWALPreamble() *WALPreamble {
	return &WALPreamble{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}
}

// LogError create a log event message from an error.
func LogError(p *Peer, s error) *Message {
	return &Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: Message_LogEvent,
		Peer: p,
		Ts:   time.Now().Unix(),
		Event: &Message_Log{
			Log: &Log{
				Log: s.Error(),
			},
		},
	}
}

func NewConnectionLog(p *Peer, s ConnectionEvent_Type, description string) *Message {
	return &Message{
		Id:        uuid.Must(uuid.NewV4()).String(),
		Type:      Message_LogEvent,
		Ephemeral: true,
		Peer:      p,
		Ts:        time.Now().Unix(),
		Event: &Message_Connection{
			Connection: &ConnectionEvent{
				State:       s,
				Description: description,
			},
		},
	}
}
func NewLogHistoryEvent(m ...*Message) *LogHistoryEvent {
	return &LogHistoryEvent{
		Messages: m,
	}
}

func NewLogHistoryMessage(p *Peer, log *LogHistoryEvent) *Message {
	return &Message{
		Id:        uuid.Must(uuid.NewV4()).String(),
		Type:      Message_LogHistoryEvent,
		Ephemeral: true,
		Peer:      p,
		Ts:        time.Now().Unix(),
		Event: &Message_History{
			History: log,
		},
	}
}

func NewLogHistoryFromMessages(p *Peer, m ...*Message) *Message {
	return NewLogHistoryMessage(p, NewLogHistoryEvent(m...))
}

// PeerEvent ...
func PeerEvent(p *Peer) *Message {
	return &Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  Message_PeerEvent,
		Peer:  p,
		Ts:    time.Now().Unix(),
		Event: &Message_None{},
	}
}

// NodeEvent ...
func NodeEvent(p *Peer, event Message_NodeEvent) *Message {
	return &Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  Message_PeerEvent,
		Peer:  p,
		Ts:    time.Now().Unix(),
		Event: &Message_Membership{Membership: event},
	}
}

func deployToOptions(d *Deploy) (dopts *DeployOptions) {
	if d.Options != nil {
		return d.Options
	}

	return &DeployOptions{}
}

func deployToArchive(d *Deploy) (a *Archive) {
	if d.Archive != nil {
		return d.Archive
	}

	return &Archive{}
}

type doption func(*DeployCommand)

func deployCommand(c DeployCommand_Command, by string, options ...doption) *DeployCommand {
	cmd := &DeployCommand{
		Initiator: by,
		Command:   c,
	}

	for _, opt := range options {
		opt(cmd)
	}

	return cmd
}

func (t *Archive) DeployOption(dc *DeployCommand) {
	dc.Archive = t
}

func (t *DeployOptions) DeployOption(dc *DeployCommand) {
	dc.Options = t
}

func updateDTS(dc *DeployCommand) {
	if dc.Archive == nil {
		return
	}

	dc.Archive.Dts = time.Now().Unix()
}

// DeployCommandBegin creates a begin deploy command.
func DeployCommandBegin(by string, a *Archive, opts *DeployOptions, options ...doption) *DeployCommand {
	return deployCommand(DeployCommand_Begin, by, append(options, a.DeployOption, opts.DeployOption)...)
}

// DeployCommandCancel create a cancellation command.
func DeployCommandCancel(by string) *DeployCommand {
	return &DeployCommand{
		Command:   DeployCommand_Cancel,
		Initiator: by,
	}
}

// DeployCommandDone ...
func DeployCommandDone(by string, options ...doption) *DeployCommand {
	return deployCommand(DeployCommand_Done, by, append(options, updateDTS)...)
}

// DeployCommandFailedQuick ...
func DeployCommandFailedQuick(options ...doption) *DeployCommand {
	return DeployCommandFailed("", options...)
}

// DeployCommandFailed ...
func DeployCommandFailed(by string, options ...doption) *DeployCommand {
	return deployCommand(DeployCommand_Failed, by, append(options, updateDTS)...)
}

// DeployCommandRestart delivered when a deploy is automatically restarting.
func DeployCommandRestart() *DeployCommand {
	return &DeployCommand{
		Command: DeployCommand_Restart,
	}
}

// DeployCommand send a deploy command message
func NewDeployCommand(p *Peer, dc *DeployCommand) *Message {
	return &Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: Message_DeployCommandEvent,
		Peer: p,
		Ts:   time.Now().Unix(),
		Event: &Message_DeployCommand{
			DeployCommand: dc,
		},
	}
}

// DeployEvent represents a deploy being triggered.
func DeployEvent(p *Peer, d *Deploy) *Message {
	return deployEvent(d.Stage, p, d.Initiator, deployToOptions(d), deployToArchive(d), "")
}

func DeployEventFailed(p *Peer, by string, di *DeployOptions, a *Archive, cause error) *Message {
	return deployEvent(Deploy_Failed, p, by, di, a, cause.Error())
}

func deployEvent(t Deploy_Stage, p *Peer, by string, di *DeployOptions, a *Archive, err string) *Message {
	return &Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  Message_DeployEvent,
		Peer:  p,
		Ts:    time.Now().Unix(),
		Event: &Message_Deploy{Deploy: &Deploy{Stage: t, Initiator: by, Options: di, Archive: a, Error: err}},
	}
}

func integerEvent(p *Peer, t Message_Type, n int64) *Message {
	return &Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: t,
		Peer: p,
		Ts:   time.Now().Unix(),
		Event: &Message_Int{
			Int: n,
		},
	}
}

// ApplyToStateMachine utility function that applies an event to the provided
// state machine handling the encoding and error handling logic.
func ApplyToStateMachine(r *raft.Raft, m *Message, d time.Duration) (err error) {
	var (
		encoded []byte
		future  raft.ApplyFuture
		ok      bool
	)

	if encoded, err = proto.Marshal(m); err != nil {
		return errors.WithStack(err)
	}

	// write the event to the WAL.
	future = r.Apply(encoded, d)

	if err = future.Error(); err != nil {
		return errors.WithStack(err)
	}

	if err, ok = future.Response().(error); ok {
		return errors.WithStack(err)
	}

	return nil
}
