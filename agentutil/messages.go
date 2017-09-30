package agentutil

import (
	"time"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

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

// DeployEvent represents a deploy being triggered.
func DeployEvent(p agent.Peer, a agent.Archive) agent.Message {
	return agent.Message{
		Type:  agent.Message_DeployEvent,
		Peer:  &p,
		Ts:    time.Now().Unix(),
		Event: &agent.Message_Archive{Archive: &a},
	}
}
