package ux

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
)

// Logging based ux
func Logging(ctx context.Context, wg *sync.WaitGroup, events chan agent.Message) {
	logger := log.New(os.Stderr, "[CLIENT] ", 0)
	defer wg.Done()
	for {
		select {
		case m := <-events:
			switch m.Type {
			case agent.Message_PeerEvent:
			case agent.Message_DeployCommandEvent:
			case agent.Message_PeersCompletedEvent:
				// Do nothing.
			case agent.Message_PeersFoundEvent:
				logger.Printf(
					"%s - INFO - located %d peers\n",
					messagePrefix(m),
					m.GetInt(),
				)
			case agent.Message_DeployEvent:
				d := m.GetDeploy()
				logger.Printf(
					"%s - Deploy %s %s\n",
					messagePrefix(m),
					bw.RandomID(d.Archive.DeploymentID),
					d.Stage,
				)
			case agent.Message_LogEvent:
				d := m.GetLog()
				logger.Printf(
					"%s - INFO - %s\n",
					messagePrefix(m),
					d.Log,
				)
			default:
				logger.Printf("%s - %s\n", messagePrefix(m), m.Type)
			}
		case _ = <-ctx.Done():
			return
		}
	}
}
