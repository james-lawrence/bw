package deployment

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/x/logx"
)

const (
	dispatchTimeout = 10 * time.Second
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

// AwaitDeployResult waits for the deployment result of the context
func AwaitDeployResult(dctx DeployContext) DeployResult {
	defer close(dctx.completed)
	return <-dctx.completed
}

// NewDeployContext create new deployment context containing configuration information
// for a single deploy.
func NewDeployContext(workdir string, p agent.Peer, dopts agent.DeployOptions, a agent.Archive, options ...DeployContextOption) (_did DeployContext, err error) {
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

	logger = dlog{log: log.New(ioutil.Discard, "", 0)}
	if !dopts.SilenceDeployLogs {
		if logfile, logger, err = newLogger(id, root, "[DEPLOY] "); err != nil {
			return _did, err
		}
	}

	dctx := DeployContext{
		Local:         p,
		ID:            id,
		Root:          root,
		ArchiveRoot:   archiveDir,
		Log:           logger,
		Archive:       a,
		DeployOptions: dopts,
		logfile:       logfile,
		dispatcher:    agentutil.LogDispatcher{},
		completed:     make(chan DeployResult),
		done:          &sync.Once{},
	}

	for _, opt := range options {
		opt(&dctx)
	}

	dctx.Log.Println("---------------------- DURATION", dctx.timeout(), "----------------------")
	dctx.deadline, dctx.cancel = context.WithTimeout(context.Background(), dctx.timeout())
	return dctx, nil
}

// DeployContext - information about the deploy, such as the root directory, the logfile, the archive etc.
type DeployContext struct {
	Local         agent.Peer
	completed     chan DeployResult
	ID            bw.RandomID
	Root          string
	ArchiveRoot   string
	Log           logger
	logfile       *os.File
	Archive       agent.Archive
	DeployOptions agent.DeployOptions
	dispatcher    dispatcher
	deadline      context.Context
	cancel        context.CancelFunc
	done          *sync.Once
}

func (t DeployContext) timeout() time.Duration {
	return time.Duration(t.DeployOptions.Timeout)
}

// Dispatch an event to the cluster
func (t DeployContext) Dispatch(m ...agent.Message) error {
	return logx.MaybeLog(dispatch(t.dispatcher, dispatchTimeout, m...))
}

// Cancel cancel the deploy.
func (t DeployContext) Cancel(reason error) {
	t.Dispatch(agentutil.LogError(t.Local, errors.Wrap(reason, "cancelled deploy")))
	t.cancel()
}

// Done is responsible for closing out the deployment context.
func (t DeployContext) Done(result error) error {
	t.done.Do(func() {
		logErr(errors.Wrap(t.logfile.Sync(), "failed to sync deployment log"))
		logErr(errors.Wrap(t.logfile.Close(), "failed to close deployment log"))

		if t.completed != nil {
			t.completed <- DeployResult{
				Error:         result,
				DeployContext: t,
			}
		}
	})

	return result
}

type logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}

func newCancelDeployContext() DeployContext {
	return DeployContext{
		dispatcher:    agentutil.DiscardDispatcher{},
		DeployOptions: agent.DeployOptions{Timeout: int64(bw.DefaultDeployTimeout)},
		completed:     make(chan DeployResult),
		cancel:        func() {},
	}
}

// DeployState represents the state of a deploy.
type DeployState struct {
	current        agent.Deploy
	currentContext DeployContext
	state          *uint32
}

func newDeployState() DeployState {
	return DeployState{
		state:          new(uint32),
		currentContext: newCancelDeployContext(),
	}
}

// DeployResult - result of a deploy.
type DeployResult struct {
	DeployContext
	Error error
}

func (t DeployResult) deployComplete() agent.Deploy {
	tmpa := t.Archive
	tmpo := t.DeployOptions
	t.Log.Println("------------------- deploy completed -------------------")
	d := agent.Deploy{Stage: agent.Deploy_Completed, Archive: &tmpa, Options: &tmpo}
	logx.MaybeLog(t.Dispatch(agentutil.DeployEvent(t.Local, d)))
	return d
}

func (t DeployResult) deployFailed(err error) agent.Deploy {
	tmpa := t.Archive
	tmpo := t.DeployOptions

	t.Log.Printf("cause:\n%+v\n", err)
	t.Log.Println("------------------- deploy failed -------------------")
	d := agent.Deploy{Stage: agent.Deploy_Failed, Archive: &tmpa, Options: &tmpo}
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
	Dispatch(context.Context, ...agent.Message) error
}
