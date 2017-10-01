package deployment

import (
	"encoding/gob"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/x/logx"
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
	// StatusUnknown the system doesn't current know its state.
	StatusUnknown StatusEnum = iota
	// StatusReady the system is willing to accept a deployment.
	StatusReady
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
	case StatusUnknown:
		return unknown{}
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

type unknown struct{}

func (t unknown) Error() string {
	return "unknown"
}

func (t unknown) Status() StatusEnum {
	return StatusUnknown
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
func AgentStateFromStatus(status Status) agent.Peer_State {
	switch status.Status() {
	case StatusReady:
		return agent.Peer_Ready
	case StatusCanary:
		return agent.Peer_Canary
	case StatusDeploying:
		return agent.Peer_Deploying
	case StatusFailed:
		return agent.Peer_Failed
	default:
		return agent.Peer_Unknown
	}
}

// AgentStateToStatus ...
func AgentStateToStatus(info agent.Peer_State) Status {
	switch info {
	case agent.Peer_Unknown:
		return unknown{}
	case agent.Peer_Ready:
		return ready{}
	case agent.Peer_Canary:
		return canary{}
	case agent.Peer_Deploying:
		return deploying{}
	default:
		return failed{}
	}
}

// IsReady returns true if the node is in a ready state.
func IsReady(err error) bool {
	return isStatus(err, StatusReady)
}

// IsUnknown returns true if the node is in a ready state.
func IsUnknown(err error) bool {
	return isStatus(err, StatusUnknown)
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
	// Deployments info about the deployment coordinator
	// idle, canary, deploying, locked, and the list of recent deployments.
	Deployments() (agent.Peer_State, []*agent.Archive, error)
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

// DeployContextOptionDispatcher ...
func DeployContextOptionDispatcher(d dispatcher) DeployContextOption {
	return func(dctx *DeployContext) {
		dctx.dispatcher = d
	}
}

// NewDeployContext ...
func NewDeployContext(workdir string, p agent.Peer, a agent.Archive, options ...DeployContextOption) (_did DeployContext, err error) {
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
		Local:      p,
		ID:         id,
		Root:       root,
		Log:        logger,
		Archive:    a,
		logfile:    logfile,
		dispatcher: logDispatcher{},
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

type dispatcher interface {
	Dispatch(...agent.Message) error
}

type logDispatcher struct{}

func (t logDispatcher) Dispatch(ms ...agent.Message) error {
	for _, m := range ms {
		log.Printf("dispatched %#v\n", m)
	}
	return nil
}

// DeployContext - information about the deploy, such as the root directory, the logfile, the archive etc.
type DeployContext struct {
	Local      agent.Peer
	ID         bw.RandomID
	Root       string
	Log        logger
	logfile    *os.File
	Archive    agent.Archive
	dispatcher dispatcher
	completed  chan DeployResult
}

// Dispatch an event to the cluster
func (t DeployContext) Dispatch(m ...agent.Message) error {
	return t.dispatcher.Dispatch(m...)
}

func (t DeployContext) deployComplete() {
	t.Log.Println("------------------- deploy completed -------------------")
	logx.MaybeLog(t.Dispatch(agentutil.DeployCompletedEvent(t.Local, t.Archive)))
}

func (t DeployContext) deployFailed(err error) {
	t.Log.Println(err)
	t.Log.Println("------------------- deploy failed -------------------")
	logx.MaybeLog(t.Dispatch(
		agentutil.LogEvent(t.Local, err.Error()),
		agentutil.DeployFailedEvent(t.Local, t.Archive),
	))
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
