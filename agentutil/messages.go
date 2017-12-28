package agentutil

import (
	"time"

	"github.com/james-lawrence/bw/agent"
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
		Type:  agent.Message_PeerEvent,
		Peer:  &p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_None{},
	}
}

func deployToArchive(d agent.Deploy) (a agent.Archive) {
	if d.Archive != nil {
		return *d.Archive
	}

	return a
}

// DeployEvent represents a deploy being triggered.
func DeployEvent(p agent.Peer, d agent.Deploy) agent.Message {
	return deployEvent(d.Stage, p, deployToArchive(d))
}

func deployEvent(t agent.Deploy_Stage, p agent.Peer, a agent.Archive) agent.Message {
	return agent.Message{
		Type:  agent.Message_DeployEvent,
		Peer:  &p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_Deploy{Deploy: &agent.Deploy{Stage: t, Archive: &a}},
	}
}

func integerEvent(p agent.Peer, t agent.Message_Type, n int64) agent.Message {
	return agent.Message{
		Type: t,
		Peer: &p,
		Ts:   time.Now().Unix(),
		Event: &agent.Message_Int{
			Int: n,
		},
	}
}
