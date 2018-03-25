package deployment

import (
	"io/ioutil"
	"log"
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

// DeployContextOption options for a DeployContext
type DeployContextOption func(dctx *DeployContext)

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

// DeployContextOptionQuiet disable logging, only used in tests.
func DeployContextOptionQuiet(quiet bool) DeployContextOption {
	return func(dctx *DeployContext) {
		if quiet {
			dctx.Log = dlog{log: log.New(ioutil.Discard, "", 0)}
		}
	}
}

// AwaitDeployResult waits for the deployment result of the context
func AwaitDeployResult(dctx DeployContext) DeployResult {
	defer close(dctx.completed)
	return <-dctx.completed
}

// NewDeployContext create new deployment context containing configuration information
// for a single deploy.
func NewDeployContext(workdir string, p agent.Peer, a agent.Archive, options ...DeployContextOption) (_did DeployContext, err error) {
	var (
		logfile *os.File
		logger  dlog
	)

	id := bw.RandomID(a.DeploymentID)
	root := filepath.Join(workdir, id.String())
	archiveDir := filepath.Join(root, "archive")
	if err = os.MkdirAll(root, 0755); err != nil {
		return _did, errors.WithMessage(err, "failed to create deployment directory")
	}

	if err = os.MkdirAll(archiveDir, 0755); err != nil {
		return _did, errors.WithMessage(err, "failed to create archive directory")
	}

	if logfile, logger, err = newLogger(id, root, "[DEPLOY] "); err != nil {
		return _did, err
	}

	dctx := DeployContext{
		Local:       p,
		ID:          id,
		Root:        root,
		ArchiveRoot: archiveDir,
		Log:         logger,
		Archive:     a,
		logfile:     logfile,
		dispatcher:  agentutil.LogDispatcher{},
		deadline:    time.Now().UTC().Add(5 * time.Minute),
		completed:   make(chan DeployResult),
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
	completed   chan DeployResult
	ID          bw.RandomID
	Root        string
	ArchiveRoot string
	Log         logger
	logfile     *os.File
	Archive     agent.Archive
	dispatcher  dispatcher
	deadline    time.Time
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
