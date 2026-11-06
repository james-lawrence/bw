package agent

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/langx"
	"github.com/james-lawrence/bw/internal/timex"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// MessageTime the time the message was created. UUIDv7 ids carry their own timestamp and Ts is
// in nanoseconds. anything else (v4 ids from older nodes) has a Ts in seconds.
// the zero time is returned when neither is available, callers must treat that as an error.
func MessageTime(m *Message) time.Time {
	switch id := uuid.FromStringOrNil(m.GetId()); id.Version() {
	case uuid.V7:
		var ts time.Time
		if m.GetTs() > 0 {
			ts = time.Unix(0, m.GetTs())
		}

		v := errorsx.Zero(uuid.TimestampFromV7(id))
		return timex.FirstNonZero(errorsx.Zero(v.Time()), ts)
	default:
		if m.GetTs() > 0 {
			return time.Unix(m.GetTs(), 0)
		}

		return time.Time{}
	}
}

// Stamp identifies the message and records when it was created from a single reading of the clock,
// so the timestamp embedded in the uuidv7 id matches Ts (to the millisecond).
func Stamp(m *Message) {
	now := time.Now()
	m.Id = uuid.Must(uuid.NewV7AtTime(now)).String()
	m.Ts = now.UnixNano()
}

// test helper
func StampZero(m *Message) {
	now := time.Time{}
	m.Id = uuid.Nil.String()
	m.Ts = now.UnixNano()
}

// PeersFoundEvent ...
func PeersFoundEvent(p *Peer, n int64) *Message {
	return integerEvent(p, Message_PeersFoundEvent, n)
}

// PeersCompletedEvent ...
func PeersCompletedEvent(p *Peer, n int64) *Message {
	return integerEvent(p, Message_PeersCompletedEvent, n)
}

// ErrDuplicate is returned when a message with the same id was already applied. senders retry until they
// see success, when a retry is answered with this the earlier attempt was delivered.
func ErrDuplicate(m *Message) error {
	return status.Errorf(codes.AlreadyExists, "duplicate message: %s", m.GetId())
}

// IsDuplicate reports whether the error is a rejection for an already applied message.
func IsDuplicate(err error) bool {
	return grpcx.IsCode(err, codes.AlreadyExists)
}

// MessageOption modifies a message as it is created.
type MessageOption func(*Message)

// MessageOptionEphemeral marks the message as ephemeral, it is not recorded in the write ahead log.
func MessageOptionEphemeral(m *Message) {
	m.Ephemeral = true
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
	return new(langx.Clone(Message{
		Type: Message_LogEvent,
		Peer: p,
		Event: &Message_Log{
			Log: &Log{
				Log: s.Error(),
			},
		},
	}, Stamp))
}

func NewConnectionLog(p *Peer, s ConnectionEvent_Type, description string) *Message {
	return new(langx.Clone(Message{
		Type:      Message_LogEvent,
		Ephemeral: true,
		Peer:      p,
		Event: &Message_Connection{
			Connection: &ConnectionEvent{
				State:       s,
				Description: description,
			},
		},
	}, Stamp))
}
func NewLogHistoryEvent(m ...*Message) *LogHistoryEvent {
	return &LogHistoryEvent{
		Messages: m,
	}
}

func NewLogHistoryMessage(p *Peer, log *LogHistoryEvent) *Message {
	return new(langx.Clone(Message{
		Type:      Message_LogHistoryEvent,
		Ephemeral: true,
		Peer:      p,
		Event: &Message_History{
			History: log,
		},
	}, Stamp))
}

func NewLogHistoryFromMessages(p *Peer, m ...*Message) *Message {
	return NewLogHistoryMessage(p, NewLogHistoryEvent(m...))
}

func NewDeployHeartbeat(p *Peer) *Message {
	return new(langx.Clone(Message{
		Type:      Message_DeployHeartbeat,
		Ephemeral: true,
		Peer:      p,
		Event: &Message_Heartbeat{
			Heartbeat: &DeployHeartbeat{},
		},
	}, Stamp))
}

// NewUploadProgress reports how much of the archive the client has uploaded.
// it is client local and must never be dispatched, the quorum's WAL rejects it.
func NewUploadProgress(p *Peer, bytes, total uint64) *Message {
	return new(langx.Clone(Message{
		Type:      Message_UploadEvent,
		Ephemeral: true,
		Peer:      p,
		Event: &Message_Upload{
			Upload: &UploadProgress{Bytes: bytes, Total: total},
		},
	}, Stamp))
}

// NewRestoreInitiated marks the start of a snapshot restore of the write ahead log. it is local to the
// node, restores are never replicated, so it is ephemeral.
func NewRestoreInitiated(p *Peer) *Message {
	return newRestore(p, Message_RestoreEventInitiated)
}

// NewRestoreCompleted marks the end of a snapshot restore, see NewRestoreInitiated.
func NewRestoreCompleted(p *Peer) *Message {
	return newRestore(p, Message_RestoreEventCompleted)
}

func newRestore(p *Peer, t Message_Type) *Message {
	return new(langx.Clone(Message{
		Type:      t,
		Ephemeral: true,
		Peer:      p,
		Event:     &Message_None{},
	}, Stamp))
}

// PeerEvent ...
func PeerEvent(p *Peer, options ...MessageOption) *Message {
	return new(langx.Clone(Message{
		Type:  Message_PeerEvent,
		Peer:  p,
		Event: &Message_None{},
	}, Stamp, langx.Compose(options...)))
}

func LogLevel(p *Peer, level Log_Level, s string, options ...MessageOption) *Message {
	return new(langx.Clone(Message{
		Type:      Message_LogEvent,
		Ephemeral: true,
		Peer:      p,
		Event: &Message_Log{
			Log: &Log{
				Level: level,
				Log:   s,
			},
		},
	}, options...))
}

// LogInfo create a log event message.
func LogInfo(p *Peer, s string, options ...MessageOption) *Message {
	return LogLevel(p, Log_INFO, s, Stamp, langx.Compose(options...))
}

func LogUser(p *Peer, s string, options ...MessageOption) *Message {
	return LogLevel(p, Log_USER, s, Stamp, langx.Compose(options...))
}

func LogWarning(p *Peer, s string) *Message {
	return LogLevel(p, Log_WARN, s, Stamp)
}

func LogDebug(p *Peer, s string) *Message {
	return LogLevel(p, Log_DEBUG, s, Stamp)
}

func LogMessage(m *Message, level Log_Level, s string) *Message_Log {
	return &Message_Log{
		Log: &Log{
			Level: level,
			Log:   s,
		},
	}
}

// NodeEvent ...
func NodeEvent(p *Peer, event Message_NodeEvent) *Message {
	return new(langx.Clone(Message{
		Type:  Message_PeerEvent,
		Peer:  p,
		Event: &Message_Membership{Membership: event},
	}, Stamp))
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

	dc.Archive.Dts = time.Now().UnixNano()
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

// DeployCommandRestart delivered when a deploy is automatically restarting. running is the deploy being
// restarted, previous the node that was coordinating it. both are optional.
func DeployCommandRestart(previous *Peer, running *DeployCommand) *DeployCommand {
	return &DeployCommand{
		Command:   DeployCommand_Restart,
		Archive:   running.GetArchive(),
		Options:   running.GetOptions(),
		Initiator: running.GetInitiator(),
		Previous:  previous,
	}
}

// DeployCommand send a deploy command message
func NewDeployCommand(p *Peer, dc *DeployCommand) *Message {
	return new(langx.Clone(Message{
		Type: Message_DeployCommandEvent,
		Peer: p,
		Event: &Message_DeployCommand{
			DeployCommand: dc,
		},
	}, Stamp))
}

// DeployEvent represents a deploy being triggered.
func DeployEvent(p *Peer, d *Deploy) *Message {
	return deployEvent(d.Stage, p, d.Initiator, deployToOptions(d), deployToArchive(d), "")
}

func DeployEventFailed(p *Peer, by string, di *DeployOptions, a *Archive, cause error) *Message {
	return deployEvent(Deploy_Failed, p, by, di, a, cause.Error())
}

func deployEvent(t Deploy_Stage, p *Peer, by string, di *DeployOptions, a *Archive, err string) *Message {
	return new(langx.Clone(Message{
		Type:  Message_DeployEvent,
		Peer:  p,
		Event: &Message_Deploy{Deploy: &Deploy{Stage: t, Initiator: by, Options: di, Archive: a, Error: err}},
	}, Stamp))
}

func integerEvent(p *Peer, t Message_Type, n int64) *Message {
	return new(langx.Clone(Message{
		Type: t,
		Peer: p,
		Event: &Message_Int{
			Int: n,
		},
	}, Stamp))
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
