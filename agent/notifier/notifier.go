package notifier

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/deployment/notifications"
)

type dialer interface {
	Dial(p agent.Peer) (agent.Client, error)
}

// New creates a notification agent from the given configuration.
func New(n ...notifications.Notifier) Notifier {
	return Notifier{
		n: n,
	}
}

// Notifier handles sending deployment notifications to various services.
type Notifier struct {
	n []notifications.Notifier
}

// Start processes notifications from the client.
func (t Notifier) Start(ctx context.Context, local, proxy agent.Peer, d dialer) {
	var (
		events = make(chan agent.Message, 5)
	)

	go agentutil.WatchEvents(local, proxy, d, events)

	for {
		select {
		case <-ctx.Done():
			return
		case m := <-events:
			switch event := m.GetEvent().(type) {
			case *agent.Message_DeployCommand:
				log.Println("deploy command received")
				dc := *event.DeployCommand
				for _, n := range t.n {
					notifyDeployCommand(n, dc)
				}
			case *agent.Message_Log:
				if m.Peer != nil && m.Peer.Name == local.Name {
					log.Println(event.Log.GetLog())
				}
			default:
				// ignore other commands.
			}
		}
	}
}

func notifyDeployCommand(n notifications.Notifier, dc agent.DeployCommand) {
	switch dc.Command {
	case agent.DeployCommand_Cancel, agent.DeployCommand_Done, agent.DeployCommand_Failed:
		if dc.Archive != nil {
			n.Notify(dc)
		}
	default:
	}
}
