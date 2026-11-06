package ux

import (
	"fmt"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/stringsx"
)

func messagePrefix(m *agent.Message) string {
	return fmt.Sprintf(
		"%s - %s - %.8s:%-12s",
		agent.MessageTime(m).Format(time.Stamp),
		stringsx.Tail(m.Id, 8),
		agent.DecodePeerID(m.Peer.Name),
		m.Peer.Ip,
	)
}
