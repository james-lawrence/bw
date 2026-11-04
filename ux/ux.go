package ux

import (
	"fmt"
	"time"

	"github.com/james-lawrence/bw/agent"
)

func messagePrefix(m *agent.Message) string {
	return fmt.Sprintf(
		"%s %s:%-12s",
		agent.MessageTime(m).Format(time.Stamp),
		agent.DecodePeerID(m.Peer.Name),
		m.Peer.Ip,
	)
}
