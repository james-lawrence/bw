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
	out := log.New(os.Stderr, "[CLIENT] ", 0)
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
				out.Printf(
					"%s - INFO - located %d peers\n",
					messagePrefix(m),
					m.GetInt(),
				)
			case agent.Message_DeployEvent:
				d := m.GetDeploy()
				out.Printf(
					"%s - %s %s\n",
					messagePrefix(m),
					d.Stage,
					bw.RandomID(d.Archive.DeploymentID),
				)
			case agent.Message_LogEvent:
				d := m.GetLog()
				out.Printf(
					"%s - INFO - %s\n",
					messagePrefix(m),
					d.Log,
				)
			default:
				out.Printf("%s - %s\n", messagePrefix(m), m.Type)
			}
		case _ = <-ctx.Done():
			return
		}
	}
}
