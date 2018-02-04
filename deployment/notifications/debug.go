package notifications

import (
	"log"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
)

// New ...
func New() *Stderr {
	return &Stderr{}
}

// Stderr - sends an message to a webhook.
type Stderr struct{}

// Notify send notification about a deploy
func (t Stderr) Notify(dc agent.DeployCommand) {
	log.Println(
		bw.RandomID(dc.Archive.DeploymentID).String(),
		"-",
		dc.Command.String(),
	)
}
