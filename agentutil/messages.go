package agentutil

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// PeersFoundEvent ...
func PeersFoundEvent(p *agent.Peer, n int64) *agent.Message {
	return integerEvent(p, agent.Message_PeersFoundEvent, n)
}

// PeersCompletedEvent ...
func PeersCompletedEvent(p *agent.Peer, n int64) *agent.Message {
	return integerEvent(p, agent.Message_PeersCompletedEvent, n)
}

// LogEvent create a log event message.
func LogEvent(p *agent.Peer, s string) *agent.Message {
	return &agent.Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: agent.Message_LogEvent,
		Peer: p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_Log{
			Log: &agent.Log{
				Log: s,
			},
		},
	}
}

// TLSEventMessage contains the generated TLS certificate for the cluster.
func TLSEventMessage(p *agent.Peer, auth, key, cert []byte) *agent.Message {
	tls := TLSEvent(auth, key, cert)
	return &agent.Message{
		Id:          uuid.Must(uuid.NewV4()).String(),
		Type:        agent.Message_TLSEvent,
		Peer:        p,
		Ts:          time.Now().Unix(),
		DisallowWAL: true,
		Hidden:      true,
		Event: &agent.Message_Credentials{
			Credentials: &tls,
		},
	}
}

// TLSEvent ...
func TLSEvent(auth, key, cert []byte) agent.TLSEvent {
	digest := md5.Sum(cert)
	return agent.TLSEvent{
		Fingerprint: hex.EncodeToString(digest[:]),
		Authority:   auth,
		Key:         key,
		Certificate: cert,
	}
}

// WALPreamble preamble message for the WAL.
func WALPreamble() *agent.WALPreamble {
	return &agent.WALPreamble{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}
}

// LogError create a log event message from an error.
func LogError(p *agent.Peer, s error) *agent.Message {
	return &agent.Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: agent.Message_LogEvent,
		Peer: p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_Log{
			Log: &agent.Log{
				Log: s.Error(),
			},
		},
	}
}

// PeerEvent ...
func PeerEvent(p *agent.Peer) *agent.Message {
	return &agent.Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  agent.Message_PeerEvent,
		Peer:  p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_None{},
	}
}

// NodeEvent ...
func NodeEvent(p *agent.Peer, event agent.Message_NodeEvent) *agent.Message {
	return &agent.Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  agent.Message_PeerEvent,
		Peer:  p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_Membership{Membership: event},
	}
}

func deployToOptions(d *agent.Deploy) (dopts *agent.DeployOptions) {
	if d.Options != nil {
		return d.Options
	}

	return &agent.DeployOptions{}
}

func deployToArchive(d *agent.Deploy) (a *agent.Archive) {
	if d.Archive != nil {
		return d.Archive
	}

	return &agent.Archive{}
}

// DeployCommandBegin creates a begin deploy command.
func DeployCommandBegin(by string, a *agent.Archive, opts *agent.DeployOptions) *agent.DeployCommand {
	return &agent.DeployCommand{
		Command:   agent.DeployCommand_Begin,
		Initiator: by,
		Archive:   a,
		Options:   opts,
	}
}

// DeployCommandCancel create a cancellation command.
func DeployCommandCancel(by string) *agent.DeployCommand {
	return &agent.DeployCommand{
		Command:   agent.DeployCommand_Cancel,
		Initiator: by,
	}
}

// DeployCommandDone ...
func DeployCommandDone() *agent.DeployCommand {
	return &agent.DeployCommand{
		Command: agent.DeployCommand_Done,
	}
}

// DeployCommandFailedQuick ...
func DeployCommandFailedQuick() *agent.DeployCommand {
	return &agent.DeployCommand{
		Command: agent.DeployCommand_Failed,
	}
}

// DeployCommandFailed ...
func DeployCommandFailed(by string, a *agent.Archive, opts *agent.DeployOptions) *agent.DeployCommand {
	return &agent.DeployCommand{
		Command:   agent.DeployCommand_Failed,
		Initiator: by,
		Archive:   a,
		Options:   opts,
	}
}

// DeployCommandRestart delivered when a deploy is automatically restarting.
func DeployCommandRestart() *agent.DeployCommand {
	return &agent.DeployCommand{
		Command: agent.DeployCommand_Restart,
	}
}

// DeployCommand send a deploy command message
func DeployCommand(p *agent.Peer, dc *agent.DeployCommand) *agent.Message {
	return &agent.Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: agent.Message_DeployCommandEvent,
		Peer: p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_DeployCommand{
			DeployCommand: dc,
		},
	}
}

// DeployEvent represents a deploy being triggered.
func DeployEvent(p *agent.Peer, d *agent.Deploy) *agent.Message {
	return deployEvent(d.Stage, p, deployToOptions(d), deployToArchive(d), "")
}

func DeployEventFailed(p *agent.Peer, di *agent.DeployOptions, a *agent.Archive, cause error) *agent.Message {
	return deployEvent(agent.Deploy_Failed, p, di, a, cause.Error())
}

func deployEvent(t agent.Deploy_Stage, p *agent.Peer, di *agent.DeployOptions, a *agent.Archive, err string) *agent.Message {
	return &agent.Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  agent.Message_DeployEvent,
		Peer:  p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_Deploy{Deploy: &agent.Deploy{Stage: t, Options: di, Archive: a, Error: err}},
	}
}

func integerEvent(p *agent.Peer, t agent.Message_Type, n int64) *agent.Message {
	return &agent.Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: t,
		Peer: p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_Int{
			Int: n,
		},
	}
}

// ApplyToStateMachine utility function that applies an event to the provided
// state machine handling the encoding and error handling logic.
func ApplyToStateMachine(r *raft.Raft, m *agent.Message, d time.Duration) (err error) {
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
