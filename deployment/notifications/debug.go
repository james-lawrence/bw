package notifications

import (
	"log"

	"github.com/james-lawrence/bw/agent"
)

// New ...
func New() *Stderr {
	return &Stderr{
		Message: "deploy ${BEARDED_WOOKIE_NOTIFICATIONS_DEPLOY_ID} - ${BEARDED_WOOKIE_NOTIFICATIONS_DEPLOY_RESULT}",
	}
}

// Stderr - sends an message to a webhook.
type Stderr struct {
	Message string
}

// Notify send notification about a deploy
func (t Stderr) Notify(dc agent.DeployCommand) {
	log.Println(ExpandEnv(t.Message, dc))
}
