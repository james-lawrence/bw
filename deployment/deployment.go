package deployment

import "encoding/gob"

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

// Status represents the current status of the coorindator.
type Status interface {
	error
	Status() StatusEnum
}

type ready struct{}

func (t ready) Error() string {
	return "coordinator is currently ready"
}

func (t ready) Status() StatusEnum {
	return StatusReady
}

type deploying struct{}

func (t deploying) Error() string {
	return "coordinator is currently deploying"
}

func (t deploying) Status() StatusEnum {
	return StatusDeploying
}

type locked struct{}

func (t locked) Error() string {
	return "coordinator is currently locked and refusing deployments"
}

func (t locked) Status() StatusEnum {
	return StatusLocked
}

type failed struct{}

func (t failed) Error() string {
	return "coordinator failed its deployment"
}

func (t failed) Status() StatusEnum {
	return StatusFailed
}

func IsReady(err error) bool {
	return isStatus(err, StatusReady)
}

func IsLocked(err error) bool {
	return isStatus(err, StatusLocked)
}

func IsDeploying(err error) bool {
	return isStatus(err, StatusDeploying)
}

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

type Coordinator interface {
	// Status of the deployment coordinator
	// idle, deploying, locked
	Status() error
	// Deploy trigger a deploy
	Deploy() error
}
