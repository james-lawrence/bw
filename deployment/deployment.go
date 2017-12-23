package deployment

import (
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/x/logx"
)

// NewStatus ...
func NewStatus(ps agent.Peer_State) error {
	return status(ps)
}

type status agent.Peer_State

func (t status) Error() string {
	switch t {
	default:
		return agent.Peer_State(t).String()
	}
}

func (t status) state() agent.Peer_State {
	return agent.Peer_State(t)
}

// IsReady returns true if the node is in a ready state.
func IsReady(c error) bool {
	return isStatus(c, agent.Peer_Ready)
}

// IsUnknown returns true if the node is in a ready state.
func IsUnknown(c error) bool {
	return isStatus(c, agent.Peer_Unknown)
}

// IsCanary returns true if the node is in a canary state.
func IsCanary(c error) bool {
	return isStatus(c, agent.Peer_Canary)
}

// IsDeploying returns true if the node is in a deploying state.
func IsDeploying(c error) bool {
	return isStatus(c, agent.Peer_Deploying)
}

// IsFailed returns true if the node is in a failed state.
func IsFailed(c error) bool {
	return isStatus(c, agent.Peer_Failed)
}

func isStatus(current error, expected agent.Peer_State) bool {
	if s, ok := current.(status); ok {
		return s.state() == expected
	}

	return false
}

func unwrapStatus(err error) agent.Peer_State {
	if s, ok := err.(status); ok {
		return s.state()
	}

	return agent.Peer_Unknown
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
	t.Log.Printf("cause:\n%+v\n", err)
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
