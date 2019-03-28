package ux

import (
	"github.com/james-lawrence/bw/agent"
)

type restart struct {
	cState
}

func (t restart) Consume(m agent.Message) consumer {
	t.cState.print(m)

	switch m.Type {
	case agent.Message_DeployCommandEvent:
		// ignore failures and cancels as restart will emit a cancel triggering failures.
		switch m.GetDeployCommand().Command {
		case agent.DeployCommand_Begin:
			return deploying{cState: t.cState}
		case agent.DeployCommand_Done: // just in case for some reason we see this.
			return nil
		}
	}

	// await next message by default
	return t
}
