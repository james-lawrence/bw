package ux

import (
	"fmt"
	"time"

	"bitbucket.org/jatone/bearded-wookie/agent"
)

func messagePrefix(m agent.Message) string {
	return fmt.Sprintf(
		"%s %s:%s",
		time.Unix(m.GetTs(), 0).Format(time.Stamp),
		m.Peer.Name,
		m.Peer.Ip,
	)
}
