package deployment

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/x/logx"
)

type deployer interface {
	Deploy(dctx DeployContext)
}

// Coordinator is in charge of coordinating deployments.
type Coordinator interface {
	// Deployments info about the deployment coordinator
	// idle, canary, deploying, locked, and the list of recent deployments.
	Deployments() ([]agent.Deploy, error)
	// Deploy trigger a deploy
	Deploy(agent.DeployOptions, agent.Archive) (agent.Deploy, error)
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

// DeployContextOptionDeadline ...
func DeployContextOptionDeadline(d time.Time) DeployContextOption {
	return func(dctx *DeployContext) {
		dctx.deadline = d
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
		Local:       p,
		ID:          id,
		Root:        root,
		ArchiveRoot: filepath.Join(root, "archive"),
		Log:         logger,
		Archive:     a,
		logfile:     logfile,
		dispatcher:  agentutil.LogDispatcher{},
		deadline:    time.Now().UTC().Add(5 * time.Minute),
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

func (t DeployResult) deployComplete() agent.Deploy {
	tmp := t.Archive
	t.Log.Println("------------------- deploy completed -------------------")
	d := agent.Deploy{Archive: &tmp, Stage: agent.Deploy_Completed}
	logx.MaybeLog(t.Dispatch(agentutil.DeployEvent(t.Local, d)))
	return d
}

func (t DeployResult) deployFailed(err error) agent.Deploy {
	tmp := t.Archive
	t.Log.Printf("cause:\n%+v\n", err)
	t.Log.Println("------------------- deploy failed -------------------")
	d := agent.Deploy{Archive: &tmp, Stage: agent.Deploy_Failed}
	logx.MaybeLog(t.Dispatch(
		agentutil.LogEvent(t.Local, err.Error()),
		agentutil.DeployEvent(t.Local, d),
	))
	return d
}

func (t DeployResult) complete() agent.Deploy {
	if t.Error != nil {
		return t.deployFailed(t.Error)
	}

	return t.deployComplete()
}

type dispatcher interface {
	Dispatch(...agent.Message) error
}

// DeployContext - information about the deploy, such as the root directory, the logfile, the archive etc.
type DeployContext struct {
	Local       agent.Peer
	ID          bw.RandomID
	Root        string
	ArchiveRoot string
	Log         logger
	logfile     *os.File
	Archive     agent.Archive
	dispatcher  dispatcher
	deadline    time.Time
	completed   chan DeployResult
}

// Dispatch an event to the cluster
func (t DeployContext) Dispatch(m ...agent.Message) error {
	return t.dispatcher.Dispatch(m...)
}

// Done is responsible for closing out the deployment context.
func (t DeployContext) Done(result error) error {
	logErr(errors.Wrap(t.logfile.Sync(), "failed to sync deployment log"))
	logErr(errors.Wrap(t.logfile.Close(), "failed to close deployment log"))

	if t.completed != nil {
		t.completed <- DeployResult{
			Error:         result,
			DeployContext: t,
		}
	}

	return result
}

type logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}
