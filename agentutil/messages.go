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

// PeerEvent ...
func PeerEvent(p agent.Peer) agent.Message {
	return agent.Message{
		Type:  agent.Message_PeerEvent,
		Peer:  &p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_None{},
	}
}

// DeployInitiatedEvent represents a deploy being triggered.
func DeployInitiatedEvent(p agent.Peer, a agent.Archive) agent.Message {
	return deployEvent(agent.Deploy_Initiated, p, a)
}

// DeployCompletedEvent represents a deploy being triggered.
func DeployCompletedEvent(p agent.Peer, a agent.Archive) agent.Message {
	return deployEvent(agent.Deploy_Completed, p, a)
}

// DeployFailedEvent represents a deploy being triggered.
func DeployFailedEvent(p agent.Peer, a agent.Archive) agent.Message {
	return deployEvent(agent.Deploy_Failed, p, a)
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
