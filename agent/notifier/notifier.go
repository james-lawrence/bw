package notifier

import (
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment/notifications"
)

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
func (t Notifier) Start(c agent.Client) error {
	var (
		events = make(chan agent.Message, 5)
		done   = make(chan struct{})
	)
	defer close(done)

	go t.background(events, done)

	return c.Watch(events)
}

func (t Notifier) background(events chan agent.Message, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		case m := <-events:
			switch event := m.GetEvent().(type) {
			case *agent.Message_DeployCommand:
				log.Println("deploy command received")
				dc := *event.DeployCommand
				for _, n := range t.n {
					notifyDeployCommand(n, dc)
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
