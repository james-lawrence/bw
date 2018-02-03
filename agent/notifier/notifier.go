package notifier

import (
	"github.com/james-lawrence/bw/agent"
)

type notifier interface {
	Notify(a agent.Archive)
}

// New creates a notification agent from the given configuration.
func New(c agent.Client, n notifier) Notifier {
	return Notifier{
		n: n,
		c: c,
	}
}

// Notifier handles sending deployment notifications to various services.
type Notifier struct {
	n notifier
	c agent.Client
}

// Start processes notifications from the client.
func (t Notifier) Start() error {
	var (
		err    error
		events = make(chan agent.Message, 5)
		failed = make(chan error)
	)

	go func() {
		if err = t.c.Watch(events); err != nil {
			failed <- err
		}
	}()

	for {
		select {
		case err = <-failed:
			// TODO gracefully restart the client.
			return err
		case m := <-events:
			switch event := m.GetEvent().(type) {
			case *agent.Message_DeployCommand:
				dc := *event.DeployCommand
				notifyDeployCommand(t.n, dc)
			default:
				// ignore other commands.
			}
		}
	}
}

// Stop gracefully shuts down the notifier.
func (t Notifier) Stop() error {
	return t.c.Close()
}

func notifyDeployCommand(n notifier, dc agent.DeployCommand) {
	switch dc.Command {
	case agent.DeployCommand_Cancel, agent.DeployCommand_Done:
		if dc.Archive != nil {
			archive := *dc.Archive
			n.Notify(archive)
		}
	default:
	}
}
