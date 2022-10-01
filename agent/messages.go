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

// DeployCommandBegin creates a begin deploy command.
func DeployCommandBegin(by string, a *Archive, opts *DeployOptions) *DeployCommand {
	return &DeployCommand{
		Command:   DeployCommand_Begin,
		Initiator: by,
		Archive:   a,
		Options:   opts,
	}
}

// DeployCommandCancel create a cancellation command.
func DeployCommandCancel(by string) *DeployCommand {
	return &DeployCommand{
		Command:   DeployCommand_Cancel,
		Initiator: by,
	}
}

// DeployCommandDone ...
func DeployCommandDone() *DeployCommand {
	return &DeployCommand{
		Command: DeployCommand_Done,
	}
}

// DeployCommandFailedQuick ...
func DeployCommandFailedQuick() *DeployCommand {
	return &DeployCommand{
		Command: DeployCommand_Failed,
	}
}

// DeployCommandFailed ...
func DeployCommandFailed(by string, a *Archive, opts *DeployOptions) *DeployCommand {
	return &DeployCommand{
		Command:   DeployCommand_Failed,
		Initiator: by,
		Archive:   a,
		Options:   opts,
	}
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
	return deployEvent(d.Stage, p, deployToOptions(d), deployToArchive(d), "")
}

func DeployEventFailed(p *Peer, di *DeployOptions, a *Archive, cause error) *Message {
	return deployEvent(Deploy_Failed, p, di, a, cause.Error())
}

func deployEvent(t Deploy_Stage, p *Peer, di *DeployOptions, a *Archive, err string) *Message {
	return &Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  Message_DeployEvent,
		Peer:  p,
		Ts:    time.Now().Unix(),
		Event: &Message_Deploy{Deploy: &Deploy{Stage: t, Options: di, Archive: a, Error: err}},
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
