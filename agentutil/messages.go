package agentutil

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
)

// PeersFoundEvent ...
func PeersFoundEvent(p agent.Peer, n int64) agent.Message {
	return integerEvent(p, agent.Message_PeersFoundEvent, n)
}

// PeersCompletedEvent ...
func PeersCompletedEvent(p agent.Peer, n int64) agent.Message {
	return integerEvent(p, agent.Message_PeersCompletedEvent, n)
}

// LogEvent create a log event message.
func LogEvent(p agent.Peer, s string) agent.Message {
	return agent.Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: agent.Message_LogEvent,
		Peer: &p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_Log{
			Log: &agent.Log{
				Log: s,
			},
		},
	}
}

// LogError create a log event message from an error.
func LogError(p agent.Peer, s error) agent.Message {
	return agent.Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: agent.Message_LogEvent,
		Peer: &p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_Log{
			Log: &agent.Log{
				Log: s.Error(),
			},
		},
	}
}

// PeerEvent ...
func PeerEvent(p agent.Peer) agent.Message {
	return agent.Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  agent.Message_PeerEvent,
		Peer:  &p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_None{},
	}
}

// NodeEvent ...
func NodeEvent(p agent.Peer, event agent.Message_NodeEvent) agent.Message {
	return agent.Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  agent.Message_PeerEvent,
		Peer:  &p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_Membership{Membership: event},
	}
}

func deployToOptions(d agent.Deploy) (dopts agent.DeployOptions) {
	if d.Options != nil {
		return *d.Options
	}

	return dopts
}

func deployToArchive(d agent.Deploy) (a agent.Archive) {
	if d.Archive != nil {
		return *d.Archive
	}

	return a
}

// DeployCommand send a deploy command message
func DeployCommand(p agent.Peer, dc agent.DeployCommand) agent.Message {
	return agent.Message{
		Type: agent.Message_DeployCommandEvent,
		Peer: &p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_DeployCommand{
			DeployCommand: &dc,
		},
	}
}

// DeployEvent represents a deploy being triggered.
func DeployEvent(p agent.Peer, d agent.Deploy) agent.Message {
	return deployEvent(d.Stage, p, deployToOptions(d), deployToArchive(d))
}

func deployEvent(t agent.Deploy_Stage, p agent.Peer, di agent.DeployOptions, a agent.Archive) agent.Message {
	return agent.Message{
		Id:    uuid.Must(uuid.NewV4()).String(),
		Type:  agent.Message_DeployEvent,
		Peer:  &p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_Deploy{Deploy: &agent.Deploy{Stage: t, Options: &di, Archive: &a}},
	}
}

func integerEvent(p agent.Peer, t agent.Message_Type, n int64) agent.Message {
	return agent.Message{
		Id:   uuid.Must(uuid.NewV4()).String(),
		Type: t,
		Peer: &p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_Int{
			Int: n,
		},
	}
}

func ApplyToStateMachine(r *raft.Raft, m agent.Message, d time.Duration) (err error) {
	var (
		encoded []byte
		future  raft.ApplyFuture
		ok      bool
	)

	if encoded, err = proto.Marshal(&m); err != nil {
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
