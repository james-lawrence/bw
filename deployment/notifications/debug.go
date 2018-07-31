package notifications

import (
	"fmt"
	"log"

	"github.com/james-lawrence/bw/agent"
)

// New ...
func New() *Stderr {
	return &Stderr{
		Message: fmt.Sprintf("deploy ${%s} - ${%s} - ${%s}", envDeployInitiator, envDeployID, envDeployResult),
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
