package ux

import (
	"fmt"
	"time"

	"github.com/james-lawrence/bw/agent"
)

func messagePrefix(m *agent.Message) string {
	return fmt.Sprintf(
		"%s %s:%s",
		agent.MessageTime(m).Format(time.Stamp),
		m.Peer.Name,
		m.Peer.Ip,
	)
}

func messageDebug(m *agent.Message) string {
	if m == nil {
		return "[None]"
	}
	return fmt.Sprintf(
		"%s %s %s %s",
		m.Id,
		m.Peer.Ip,
		agent.MessageTime(m).Format(time.Stamp),
		m.Type,
	)
}
