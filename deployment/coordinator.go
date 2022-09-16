package deployment

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/james-lawrence/bw/internal/logx"
	"github.com/james-lawrence/bw/storage"
)

const (
	coordinaterWaiting   = 0
	coordinatorDeploying = 1
)

// New Builds a deployment Coordinator.
func New(local *agent.Peer, d deployer, options ...CoordinatorOption) Coordinator {
	const (
		defaultKeepN = 3
	)

	coord := Coordinator{
		local:    local,
		deployer: d,
		m:        &sync.Mutex{},
		dlreg:    storage.New(),
		ds:       newDeployState(),
	}

	// default options.
	CoordinatorOptionRoot(os.TempDir())(&coord)
	CoordinatorOptionKeepN(defaultKeepN)(&coord)
	CoordinatorOptionDispatcher(agentutil.DiscardDispatcher{})(&coord)

	return CloneCoordinator(coord, options...)
}

// CloneCoordinator build a coordinator from a pre-existing instance.
func CloneCoordinator(c Coordinator, options ...CoordinatorOption) Coordinator {
	dup := c
	for _, opt := range options {
		opt(&dup)
	}

	return dup
}

// CoordinatorOption options for the deployment coordinator.
type CoordinatorOption func(*Coordinator)

// CoordinatorOptionRoot - root directory for performing deployments.
func CoordinatorOptionRoot(root string) CoordinatorOption {
	return func(d *Coordinator) {
		d.root = root
		d.deploysRoot = bw.DeployDir(root)
	}
}

// CoordinatorOptionKeepN the number of previous deploys to keep.
func CoordinatorOptionKeepN(n int) CoordinatorOption {
	return func(d *Coordinator) {
		d.keepN = n
		d.cleanup = agentutil.KeepOldestN(n)
	}
}

// CoordinatorOptionDispatcher sets the dispatcher for the coordinator.
func CoordinatorOptionDispatcher(di dispatcher) CoordinatorOption {
	return func(d *Coordinator) {
		d.dispatcher = di
	}
}

// CoordinatorOptionDeployResults set the channel to send deploy results to.
func CoordinatorOptionDeployResults(dst chan *DeployResult) CoordinatorOption {
	return func(d *Coordinator) {
		d.completedObserver = dst
	}
}

// CoordinatorOptionStorage set the storage registry.
func CoordinatorOptionStorage(reg storage.DownloadFactory) CoordinatorOption {
	return func(d *Coordinator) {
		d.dlreg = reg
	}
}

// Coordinator for a deploy
type Coordinator struct {
	keepN             int // never set manually. always set by CoordinatorOptionKeepN
	root              string
	deploysRoot       string // never set manually. always set by CoordinatorOptionRoot
	local             *agent.Peer
	deployer          deployer
	dispatcher        dispatcher
	dlreg             storage.DownloadFactory
	cleanup           agentutil.Cleaner // never set manually. always set by CoordinatorOptionKeepN
	completedObserver chan *DeployResult
	ds                *DeployState
	m                 *sync.Mutex
}

func (t *Coordinator) background(dctx *DeployContext) {
	defer close(dctx.completed)

	done := <-dctx.completed
	if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
		log.Println("deployment completed", dctx.ID.String())
	}
	d := done.complete()

	if err := writeDeployMetadata(done.Root, d); err != nil {
		log.Println("failed to write deploy metadata", err)
	}

	// by default keep the oldest deploys. if we have a successful deploy then keep the newest.
	switch d.Stage {
	case agent.Deploy_Completed:
		t.update(dctx, d, agentutil.KeepNewestN(t.keepN))
	default:
		t.update(dctx, d, agentutil.KeepOldestN(t.keepN))
	}

	if t.completedObserver != nil {
		t.completedObserver <- done
	}

	atomic.SwapUint32(t.ds.state, coordinaterWaiting)
}

// Deployments about the state of the agent.
func (t *Coordinator) Deployments() (deployments []*agent.Deploy, err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if deployments, err = readAllDeployMetadata(t.deploysRoot); err != nil {
		return deployments, err
	}

	sort.Slice(deployments, func(i int, j int) bool {
		return less(deployments[i], deployments[j])
	})

	return deployments, errors.Wrap(t.correctLatestDeploy(deployments...), "failed correcting deploy records")
}

// Reset the coordinator - remove all deploys that are not successfully completed.
func (t *Coordinator) Reset() error {
	var (
		err         error
		deployments []*agent.Deploy
	)

	t.m.Lock()
	defer t.m.Unlock()

	if deployments, err = readAllDeployMetadata(t.deploysRoot); err != nil {
		return err
	}

	for _, d := range deployments {
		if d.Stage == agent.Deploy_Completed {
			continue
		}

		s := filepath.Join(t.deploysRoot, bw.RandomID(d.Archive.DeploymentID).String())

		if err = os.RemoveAll(s); err != nil {
			return err
		}
	}

	return nil
}

// Deploy deploy a given archive.
func (t *Coordinator) Deploy(ctx context.Context, opts *agent.DeployOptions, archive *agent.Archive) (d *agent.Deploy, err error) {
	var (
		ok   bool
		dctx *DeployContext
	)

	// set the timestamp of the archive to as this marks the time the archive was actually deployed.
	// and ensures it shows up properly in the deployments history
	archive.Dts = time.Now().UTC().Unix()

	// cleanup workspace directory prior to deployment. this leaves the last deployment
	// is available until the next run for debugging.
	// IMPORTANT: torrent storage relies on this behaviour in order to prevent
	// downloads from becoming permanently blocked waiting for the archive to be downloaded.
	// without this behaviour the torrent can be removed while nodes are still trying to deploy.
	// preventing further deploys.
	if soft := agentutil.MaybeClean(t.cleanup)(agentutil.Dirs(t.deploysRoot)); soft != nil {
		soft = logx.MaybeLog(errors.Wrap(soft, "failed to clear workspace directory"))
		agentutil.Dispatch(t.dispatcher, agentutil.LogError(t.local, soft))
	}

	dcopts := []DeployContextOption{
		DeployContextOptionDispatcher(t.dispatcher),
	}

	if dctx, err = NewRemoteDeployContext(ctx, t.deploysRoot, t.local, opts, archive, dcopts...); err != nil {
		agentutil.Dispatch(t.dispatcher, agentutil.LogError(t.local, err))
		return t.ds.current, err
	}

	if dctx.Log == nil || dctx.Archive.Peer == nil {
		log.Println("debug", dctx.Log == nil, spew.Sdump(dctx.Archive))
	}

	dctx.Log.Printf("deploy recieved: deployID(%s) primary(%s) location(%s)\n", dctx.ID, dctx.Archive.Peer.Name, dctx.Archive.Location)
	go func() {
		defer dctx.Log.Printf("deploy complete: deployID(%s) primary(%s) location(%s)\n", dctx.ID, dctx.Archive.Peer.Name, dctx.Archive.Location)
		t.background(dctx)
	}()

	if ok = atomic.CompareAndSwapUint32(t.ds.state, coordinaterWaiting, coordinatorDeploying); !ok {
		err = errors.Errorf("already deploying - unknown deployment - %s", t.ds.current.Stage)
		if t.ds.current.Archive != nil {
			err = errors.Errorf("%s is already deploying: %s - %s", t.ds.current.Archive.Initiator, bw.RandomID(t.ds.current.Archive.DeploymentID).String(), t.ds.current.Stage)
		}

		dctx.Dispatch(agentutil.LogError(t.local, err))
		return t.ds.current, dctx.Done(err)
	}

	d = &agent.Deploy{Archive: archive, Options: opts, Stage: agent.Deploy_Deploying}
	t.update(dctx, d, agentutil.KeepOldestN(t.keepN))

	if err = writeDeployMetadata(dctx.Root, d); err != nil {
		dctx.Dispatch(agentutil.LogError(t.local, err))
		return d, dctx.Done(errors.WithStack(err))
	}

	dctx.Dispatch(agentutil.DeployEvent(dctx.Local, d))

	if err = downloadArchive(t.dlreg, dctx); err != nil {
		return d, dctx.Done(err)
	}

	t.deployer.Deploy(dctx)

	return d, nil
}

// Logs return the logs for the given deployment ID.
func (t *Coordinator) Logs(did []byte) (logs io.ReadCloser) {
	var (
		err error
	)

	p := filepath.Join(t.deploysRoot, bw.RandomID(did).String(), bw.DeployLog)
	if logs, err = os.Open(p); err != nil {
		return io.NopCloser(iox.ErrReader(errors.Wrapf(err, "unable to open logfile %s", p)))
	}

	return logs
}

// Cancel ...
func (t *Coordinator) Cancel() {
	t.m.Lock()
	defer t.m.Unlock()
	log.Println("cancelling deploy", *t.ds.state == coordinatorDeploying)
	if ok := atomic.CompareAndSwapUint32(t.ds.state, coordinatorDeploying, coordinaterWaiting); ok {
		t.ds.currentContext.Cancel(errors.New("deploy cancel signal received"))
		agentutil.Dispatch(t.dispatcher, agentutil.LogEvent(t.local, "cancelled deploy"))
	} else {
		log.Println("ignored cancel not deploying", *t.ds.state == coordinatorDeploying)
	}
}

func (t *Coordinator) update(dctx *DeployContext, d *agent.Deploy, c agentutil.Cleaner) *agent.Deploy {
	t.m.Lock()
	defer t.m.Unlock()

	t.ds.current = d
	t.ds.currentContext = dctx
	t.cleanup = c

	return d
}

func (t *Coordinator) correctLatestDeploy(deploys ...*agent.Deploy) error {
	if len(deploys) == 0 {
		return nil
	}

	d := deploys[0]

	if d.Stage != agent.Deploy_Deploying {
		return nil
	}

	if atomic.LoadUint32(t.ds.state) == coordinatorDeploying {
		return nil
	}

	_, root, _ := deployDirs(t.deploysRoot, d.Archive)
	d.Stage = agent.Deploy_Failed
	d.Error = "dead deploy detected"

	return writeDeployMetadata(root, d)
}

func downloadArchive(dlreg storage.DownloadFactory, dctx *DeployContext) (err error) {
	var (
		dst *os.File
	)

	dctx.Log.Println("download initiated", dctx.Archive.Location)
	if dst, err = os.OpenFile(dctx.ArchiveFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600); err != nil {
		return errors.Wrap(err, "unable to open archive file")
	}

	defer func() {
		logx.MaybeLog(errors.Wrap(errorsx.Compact(dst.Sync(), dst.Close()), "archive cleanup failed"))
	}()

	if _, err = io.Copy(dst, dlreg.New(dctx.Archive.Location).Download(dctx.deadline, dctx.Archive)); err != nil {
		return errors.Wrapf(err, "retrieve archive")
	}

	if err = iox.Rewind(dst); err != nil {
		return errors.Wrap(err, "unable to rewind archive")
	}

	if err = archive.Unpack(dctx.ArchiveRoot, dst); err != nil {
		return errors.Wrapf(err, "unpack archive")
	}

	dctx.Log.Println("download completed", dctx.Archive.Location)
	return nil
}

// ResultBus bus for deploy results.
func ResultBus(in chan *DeployResult, out ...chan *DeployResult) {
	for result := range in {
		for _, dst := range out {
			dst <- result
		}
	}
}
