package agent

import (
	"time"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

// MessageBuilder ...
type MessageBuilder struct {
	agent.Peer
}

// LogEvent ...
func (t MessageBuilder) LogEvent(s string) agent.Message {
	return agent.Message{
		Type: agent.Message_LogEvent,
		Peer: &t.Peer,
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
