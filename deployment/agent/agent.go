package agent

import (
	"time"
)

// MessageBuilder ...
type MessageBuilder struct {
	Peer
}

// LogEvent ...
func (t MessageBuilder) LogEvent(s string) *Message {
	return &Message{
		Type: Message_LogEvent,
		Peer: &t.Peer,
		Ts:   time.Now().Unix(),
		Event: &Message_Log{
			Log: &Log{
				Log: s,
			},
		},
	}
}

// PeerEvent ...
func PeerEvent(p Peer) *Message {
	return &Message{
		Type:  Message_PeerEvent,
		Peer:  &p,
		Ts:    time.Now().Unix(),
		Event: &Message_None{},
	}
}
