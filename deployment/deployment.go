package deployment

import (
	"encoding/gob"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

func init() {
	gob.Register(ready{})
	gob.Register(canary{})
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
	// StatusCanary the system is currently locked this will
	// cause it to ignore any deployment requests.
	StatusCanary
	// StatusFailed the system failed to deploy.
	StatusFailed
)

// NewStatus ...
func NewStatus(s StatusEnum) Status {
	switch s {
	case StatusReady:
		return ready{}
	case StatusDeploying:
		return deploying{}
	case StatusCanary:
		return canary{}
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

type canary struct{}

func (t canary) Error() string {
	return "locked and refusing deployments"
}

func (t canary) Status() StatusEnum {
	return StatusCanary
}

type failed struct{}

func (t failed) Error() string {
	return "failed"
}

func (t failed) Status() StatusEnum {
	return StatusFailed
}

// AgentStateFromStatus ...
func AgentStateFromStatus(status Status) agent.AgentInfo_State {
	switch status.Status() {
	case StatusReady:
		return agent.AgentInfo_Ready
	case StatusCanary:
		return agent.AgentInfo_Canary
	case StatusDeploying:
		return agent.AgentInfo_Deploying
	default:
		return agent.AgentInfo_Failed
	}
}

// AgentStateToStatus ...
func AgentStateToStatus(info agent.AgentInfo_State) Status {
	switch info {
	case agent.AgentInfo_Ready:
		return ready{}
	case agent.AgentInfo_Canary:
		return canary{}
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

// IsCanary returns true if the node is in a canary state.
func IsCanary(err error) bool {
	return isStatus(err, StatusCanary)
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
	Deploy(dctx DeployContext) error
}

// Coordinator is in charge of coordinating deployments.
type Coordinator interface {
	// Info about the deployment coordinator
	// idle, canary, deploying, locked, and the list of recent deployments.
	Info() (agent.AgentInfo, error)
	// Deploy trigger a deploy
	Deploy(a *agent.Archive) error
}

// DeployContextOption options for a DeployContext
type DeployContextOption func(dctx *DeployContext)

// DeployContextOptionCompleted allows sending a signal that the deploy completed.
func DeployContextOptionCompleted(completed chan DeployResult) DeployContextOption {
	return func(dctx *DeployContext) {
		dctx.completed = completed
	}
}

// NewDeployContext ...
func NewDeployContext(workdir string, a agent.Archive, options ...DeployContextOption) (_did DeployContext, err error) {
	var (
		logfile *os.File
		logger  dlog
	)

	id := bw.RandomID(a.DeploymentID)
	root := filepath.Join(workdir, id.String())
	if err = os.MkdirAll(root, 0755); err != nil {
		return _did, errors.WithMessage(err, "failed to create deployment directory")
	}

	if logfile, logger, err = newLogger(id, root, "[DEPLOY] "); err != nil {
		return _did, err
	}

	dctx := DeployContext{
		ID:      id,
		Root:    root,
		Log:     logger,
		Archive: a,
		logfile: logfile,
	}

	for _, opt := range options {
		opt(&dctx)
	}

	return dctx, nil
}

// DeployResult - result of a deploy.
type DeployResult struct {
	DeployContext
	Error error
}

// DeployContext - information about the deploy, such as the root directory, the logfile, the archive etc.
type DeployContext struct {
	ID        bw.RandomID
	Root      string
	Log       logger
	logfile   *os.File
	Archive   agent.Archive
	completed chan DeployResult
}

// Done is responsible for closing out the deployment context.
func (t DeployContext) Done(result error) {
	logErr(errors.Wrap(t.logfile.Sync(), "failed to sync deployment log"))
	logErr(errors.Wrap(t.logfile.Close(), "failed to close deployment log"))

	if t.completed != nil {
		t.completed <- DeployResult{
			Error:         result,
			DeployContext: t,
		}
	}
}

type logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}
