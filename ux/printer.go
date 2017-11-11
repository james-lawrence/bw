package ux

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
)

// Logging based ux
func Logging(ctx context.Context, wg *sync.WaitGroup, events chan agent.Message) {
	defer wg.Done()
	for {
		select {
		case m := <-events:
			switch m.Type {
			case agent.Message_PeersFoundEvent, agent.Message_PeersCompletedEvent:
				log.Printf(
					"%s %s:%s - %s: %d\n",
					time.Unix(m.GetTs(), 0).Format(time.Stamp),
					m.Peer.Name,
					m.Peer.Ip,
					m.Type,
					m.GetInt(),
				)
			case agent.Message_PeerEvent:
				log.Printf(
					"%s - %s: %s\n",
					messagePrefix(m),
					m.Type,
					m.Peer.Status,
				)
			case agent.Message_DeployEvent:
				d := m.GetDeploy()
				log.Printf(
					"%s - Deploy %s %s\n",
					messagePrefix(m),
					bw.RandomID(d.Archive.DeploymentID),
					d.Stage,
				)
			case agent.Message_LogEvent:
				d := m.GetLog()
				log.Printf(
					"%s %s - %s\n",
					messagePrefix(m),
					m.Type,
					d.Log,
				)
			default:
				log.Printf("%s - %s\n", messagePrefix(m), m.Type)
			}
		case _ = <-ctx.Done():
			return
		}
	}
}
