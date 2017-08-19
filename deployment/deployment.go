package deployment

import (
	"encoding/gob"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

func init() {
	gob.Register(ready{})
	gob.Register(locked{})
	gob.Register(deploying{})
	gob.Register(failed{})
}

// StatusEnum represents the state of the deployment coordinator on the local server.
type StatusEnum int

const (
	// StatusReady the system is willing to accept a deployment.
	StatusReady StatusEnum = iota
	// StatusDeploying the system is currently deploying.
	StatusDeploying
	// StatusLocked the system is currently locked this will
	// cause it to ignore any deployment requests.
	StatusLocked
	// StatusFailed the system failed to deploy.
	StatusFailed
)

func NewStatus(s StatusEnum) Status {
	switch s {
	case StatusReady:
		return ready{}
	case StatusDeploying:
		return deploying{}
	case StatusLocked:
		return locked{}
	default:
		return failed{}
	}
}

// Status represents the current status of the coorindator.
type Status interface {
	error
	Status() StatusEnum
}

type ready struct{}

func (t ready) Error() string {
	return "ready"
}

func (t ready) Status() StatusEnum {
	return StatusReady
}

type deploying struct{}

func (t deploying) Error() string {
	return "deploying"
}

func (t deploying) Status() StatusEnum {
	return StatusDeploying
}

type locked struct{}

func (t locked) Error() string {
	return "locked and refusing deployments"
}

func (t locked) Status() StatusEnum {
	return StatusLocked
}

type failed struct{}

func (t failed) Error() string {
	return "failed"
}

func (t failed) Status() StatusEnum {
	return StatusFailed
}

func AgentStateFromStatus(status Status) agent.AgentInfo_State {
	switch status.Status() {
	case StatusReady:
		return agent.AgentInfo_Ready
	case StatusLocked:
		return agent.AgentInfo_Locked
	case StatusDeploying:
		return agent.AgentInfo_Deploying
	default:
		return agent.AgentInfo_Failed
	}
}

func AgentStateToStatus(info agent.AgentInfo_State) Status {
	switch info {
	case agent.AgentInfo_Ready:
		return ready{}
	case agent.AgentInfo_Locked:
		return locked{}
	case agent.AgentInfo_Deploying:
		return deploying{}
	default:
		return failed{}
	}
}

// IsReady returns true if the node is in a ready state.
func IsReady(err error) bool {
	return isStatus(err, StatusReady)
}

// IsLocked returns true if the node is in a locked state.
func IsLocked(err error) bool {
	return isStatus(err, StatusLocked)
}

// IsDeploying returns true if the node is in a deploying state.
func IsDeploying(err error) bool {
	return isStatus(err, StatusDeploying)
}

// IsFailed returns true if the node is in a failed state.
func IsFailed(err error) bool {
	return isStatus(err, StatusFailed)
}

func isStatus(err error, expected StatusEnum) bool {
	switch err := err.(type) {
	case Status:
		if err.Status() == expected {
			return true
		}
	}

	return false
}

type deployer interface {
	Deploy(archive *agent.Archive, completed chan error) error
}

// Coordinator is in charge of coordinating deployments.
type Coordinator interface {
	// Status of the deployment coordinator
	// idle, deploying, locked
	Status() error
	// Deploy trigger a deploy
	Deploy(*agent.Archive) error
}
