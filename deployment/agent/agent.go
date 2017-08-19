package agent

import (
	"net"
	"time"
)

// MessageBuilder ...
type MessageBuilder struct {
	Node net.Addr
}

// NewLogEvent ...
func (t MessageBuilder) NewLogEvent(s string) *Message {
	return &Message{
		Type: Message_CommandInfo,
		Node: t.Node.String(),
		Ts:   time.Now().Unix(),
		Event: &Message_LogEvent{
			LogEvent: &Log{
				Log: s,
			},
		},
	}
}
