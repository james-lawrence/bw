package deployment

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
)

type deployer interface {
	Deploy(dctx *DeployContext)
}

// DeployContextOption options for a DeployContext
type DeployContextOption func(dctx *DeployContext)

// DeployContextOptionDispatcher ...
func DeployContextOptionDispatcher(d dispatcher) DeployContextOption {
	return func(dctx *DeployContext) {
		dctx.dispatcher = d
	}
}

// DeployContextOptionLog set the logger for the deployment.
func DeployContextOptionLog(l logger) DeployContextOption {
	return func(dctx *DeployContext) {
		dctx.Log = l
	}
}

// DeployContextOptionArchiveRoot set the root directory of the archive.
func DeployContextOptionArchiveRoot(ar string) DeployContextOption {
	return func(dctx *DeployContext) {
		dctx.ArchiveRoot = ar
	}
}

// DeployContextOptionTempRoot set the parent of the temp directory.
func DeployContextOptionTempRoot(ar string) DeployContextOption {
	return func(dctx *DeployContext) {
		dctx.TempRoot = ar
	}
}

// DeployContextOptionCacheRoot set the parent of the cache directory.
func DeployContextOptionCacheRoot(ar string) DeployContextOption {
	return func(dctx *DeployContext) {
		dctx.CacheRoot = ar
	}
}

// DeployContextOptionDisableReset disables resetting the root directory.
func DeployContextOptionDisableReset(dctx *DeployContext) {
	dctx.disableReset = true
}

// AwaitDeployResult waits for the deployment result of the context
func AwaitDeployResult(dctx *DeployContext) *DeployResult {
	defer close(dctx.completed)
	return <-dctx.completed
}

// NewDeployContext create a deployment context from the provided settings.
func NewDeployContext(ctx context.Context, root string, p *agent.Peer, by string, dopts *agent.DeployOptions, a *agent.Archive, options ...DeployContextOption) (_did *DeployContext, err error) {
	id := bw.RandomID(a.DeploymentID)

	dctx := &DeployContext{
		Local:         p,
		ID:            id,
		Initiator:     by,
		Root:          root,
		ArchiveRoot:   root,
		TempRoot:      root,
		CacheRoot:     root,
		ArchiveFile:   filepath.Join(root, bw.ArchiveFile),
		MetadataFile:  filepath.Join(root, deployMetadataName),
		LogFile:       filepath.Join(root, bw.DeployLog),
		Log:           dlog{Logger: log.New(io.Discard, "", 0)},
		Archive:       a,
		DeployOptions: dopts,
		dispatcher:    agentutil.DiscardDispatcher{},
		completed:     make(chan *DeployResult),
		done:          &sync.Once{},
	}

	for _, opt := range options {
		opt(dctx)
	}

	// resets the directory to prepare for the deploy.
	if err = dctx.reset(); err != nil {
		return dctx, err
	}

	if err = os.MkdirAll(dctx.ArchiveRoot, 0755); err != nil {
		return dctx, errors.WithMessage(err, "failed to create archive directory")
	}

	dctx.deadline, dctx.cancel = context.WithTimeout(ctx, dctx.timeout())

	return dctx, nil
}

func deployDirs(root string, a *agent.Archive) (bw.RandomID, string, string) {
	id := bw.RandomID(a.DeploymentID)
	droot := filepath.Join(root, id.String())
	archiveDir := filepath.Join(droot, bw.DirArchive)
	return id, droot, archiveDir
}

// NewRemoteDeployContext create new deployment context containing configuration information
// for a single deploy.
func NewRemoteDeployContext(ctx context.Context, workdir string, p *agent.Peer, by string, dopts *agent.DeployOptions, a *agent.Archive, options ...DeployContextOption) (_did *DeployContext, err error) {
	var (
		logger dlog
	)

	id, root, archiveDir := deployDirs(workdir, a)

	if err = os.MkdirAll(root, 0755); err != nil {
		return _did, errors.WithMessage(err, "failed to create deployment directory")
	}

	logger = dlog{Logger: log.New(io.Discard, "", 0)}
	if !dopts.SilenceDeployLogs {
		if logger, err = newLogger(id, root, fmt.Sprintf("[DEPLOY] [%s] ", id)); err != nil {
			return _did, err
		}
	}

	return NewDeployContext(ctx, root, p, by, dopts, a, append([]DeployContextOption{
		DeployContextOptionLog(logger),
		DeployContextOptionArchiveRoot(archiveDir),
	}, options...)...)
}

// DeployContext - information about the deploy, such as the root directory, the logfile, the archive etc.
type DeployContext struct {
	Local         *agent.Peer
	completed     chan *DeployResult
	ID            bw.RandomID
	disableReset  bool
	Initiator     string
	Root          string
	ArchiveFile   string
	ArchiveRoot   string
	TempRoot      string
	CacheRoot     string
	MetadataFile  string
	LogFile       string
	Log           logger
	Archive       *agent.Archive
	DeployOptions *agent.DeployOptions
	dispatcher    dispatcher
	deadline      context.Context
	cancel        context.CancelFunc
	done          *sync.Once
}

func (t DeployContext) timeout() time.Duration {
	return time.Duration(t.DeployOptions.Timeout)
}

// Dispatch an event to the cluster
func (t DeployContext) Dispatch(m ...*agent.Message) error {
	return agentutil.ReliableDispatch(t.deadline, t.dispatcher, m...)
}

// Cancel cancel the deploy.
func (t DeployContext) Cancel(reason error) {
	if t.deadline == nil || t.cancel == nil {
		log.Println("unable to cancel, invalid deploy state seems invalid")
		return
	}

	errorsx.Log(t.Dispatch(agent.LogError(t.Local, errors.Wrap(reason, "cancelled deploy"))))
	t.cancel()
}

// Done is responsible for closing out the deployment context.
func (t DeployContext) Done(result error) error {
	t.done.Do(func() {
		errorsx.Log(errors.Wrap(t.Log.Close(), "failed to close deployment log"))

		if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
			log.Println("deployment done", t.completed != nil)
		}
		if t.completed != nil {
			t.completed <- &DeployResult{
				Error:         result,
				DeployContext: t,
			}
		}
	})

	return result
}

func (t DeployContext) reset() (err error) {
	if t.disableReset {
		return nil
	}

	if err = os.RemoveAll(t.ArchiveRoot); err != nil {
		return errors.WithMessage(err, "failed to clear archive directory")
	}

	if err = os.Truncate(t.LogFile, 0); err != nil && !os.IsNotExist(err) {
		return errors.WithMessage(err, "failed to reset log file")
	}

	if err = os.Remove(t.MetadataFile); err != nil && !os.IsNotExist(err) {
		return errors.WithMessage(err, "failed to reset metadata file")
	}

	return nil
}

type logger interface {
	Output(int, string) error
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
	Close() error
	Write([]byte) (int, error)
}

func newCancelDeployContext() *DeployContext {
	return &DeployContext{
		dispatcher:    agentutil.DiscardDispatcher{},
		DeployOptions: &agent.DeployOptions{Timeout: int64(bw.DefaultDeployTimeout)},
		completed:     make(chan *DeployResult),
		cancel:        func() {},
	}
}

// DeployState represents the state of a deploy.
type DeployState struct {
	current        *agent.Deploy
	currentContext *DeployContext
	state          *uint32
}

func newDeployState() *DeployState {
	return &DeployState{
		state:          new(uint32),
		currentContext: newCancelDeployContext(),
	}
}

// DeployResult - result of a deploy.
type DeployResult struct {
	DeployContext
	Error error
}

func (t DeployResult) deployComplete() *agent.Deploy {
	tmpa := t.Archive
	tmpo := t.DeployOptions
	t.Log.Println("------------------- deploy completed -------------------")
	d := &agent.Deploy{Stage: agent.Deploy_Completed, Archive: tmpa, Options: tmpo}
	errorsx.Log(t.Dispatch(agent.DeployEvent(t.Local, d)))
	return d
}

func (t DeployResult) deployFailed(err error) *agent.Deploy {
	tmpa := t.Archive
	tmpo := t.DeployOptions

	t.Log.Printf("cause:\n%+v\n", err)
	t.Log.Println("------------------- deploy failed -------------------")
	d := &agent.Deploy{Stage: agent.Deploy_Failed, Archive: tmpa, Options: tmpo}
	errorsx.Log(
		t.Dispatch(
			agent.LogError(t.Local, err),
			agent.DeployEvent(t.Local, d),
		),
	)
	return d
}

func (t DeployResult) complete() *agent.Deploy {
	if t.Error != nil {
		return t.deployFailed(t.Error)
	}

	return t.deployComplete()
}

type dispatcher interface {
	Dispatch(context.Context, ...*agent.Message) error
}

func heartbeat(ctx context.Context, local *agent.Peer, freq rate.Limit, d dispatcher) {
	var (
		err error
	)

	l := rate.NewLimiter(freq, 1)

	for err = l.Wait(ctx); err == nil; err = l.Wait(ctx) {
		errorsx.Log(agentutil.Dispatch(ctx, d, agent.NewDeployHeartbeat(local)))
	}

	errorsx.Log(errors.Wrap(err, "heartbeat ending"))
}
