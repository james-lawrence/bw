package deployment

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/storage"
	"github.com/james-lawrence/bw/x/logx"
)

const (
	coordinaterWaiting   = 0
	coordinatorDeploying = 1
)

// New Builds a deployment Coordinator.
func New(local agent.Peer, d deployer, options ...CoordinatorOption) Coordinator {
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
	CoordinatorOptionDispatcher(agentutil.LogDispatcher{})(&coord)

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
		d.deploysRoot = filepath.Join(root, bw.DirDeploys)
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
func CoordinatorOptionDeployResults(dst chan DeployResult) CoordinatorOption {
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
	local             agent.Peer
	deployer          deployer
	dispatcher        dispatcher
	dlreg             storage.DownloadFactory
	cleanup           agentutil.Cleaner // never set manually. always set by CoordinatorOptionKeepN
	completedObserver chan DeployResult
	ds                DeployState
	m                 *sync.Mutex
}

func (t *Coordinator) background(dctx DeployContext) {
	defer close(dctx.completed)

	done := <-dctx.completed
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
func (t *Coordinator) Deployments() (deployments []agent.Deploy, err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if deployments, err = readAllDeployMetadata(t.deploysRoot); err != nil {
		return deployments, err
	}

	sort.Slice(deployments, func(i int, j int) bool {
		a, b := deployments[i], deployments[j]
		if a.Archive == nil {
			return true
		}

		if b.Archive == nil {
			return false
		}

		return a.Archive.Ts > b.Archive.Ts
	})

	return deployments, nil
}

// Deploy deploy a given archive.
func (t *Coordinator) Deploy(opts agent.DeployOptions, archive agent.Archive) (d agent.Deploy, err error) {
	var (
		ok   bool
		dctx DeployContext
	)

	// cleanup workspace directory prior to deployment. this leaves the last deployment
	// is available until the next run for debugging.
	// IMPORTANT: torrent storage relies on this behaviour in order to prevent
	// downloads from becoming permanently blocked waiting for the archive to be downloaded.
	// without this behaviour the torrent can be removed while nodes are still trying to deploy.
	// preventing further deploys.
	if soft := agentutil.MaybeClean(t.cleanup)(agentutil.Dirs(t.deploysRoot)); soft != nil {
		soft = errors.Wrap(soft, "failed to clear workspace directory")
		t.dispatcher.Dispatch(context.Background(), agentutil.LogEvent(t.local, soft.Error()))
		log.Println(soft)
	}

	dcopts := []DeployContextOption{
		DeployContextOptionDispatcher(t.dispatcher),
	}

	if dctx, err = NewDeployContext(t.deploysRoot, t.local, opts, archive, dcopts...); err != nil {
		logx.MaybeLog(dispatch(t.dispatcher, dispatchTimeout, agentutil.LogEvent(t.local, err.Error())))
		return t.ds.current, err
	}
	go t.background(dctx)

	if ok = atomic.CompareAndSwapUint32(t.ds.state, coordinaterWaiting, coordinatorDeploying); !ok {
		err = errors.Errorf("already deploying - unknown deployment - %s", t.ds.current.Stage)
		if t.ds.current.Archive != nil {
			err = errors.Errorf("%s is already deploying: %s - %s", t.ds.current.Archive.Initiator, bw.RandomID(t.ds.current.Archive.DeploymentID).String(), t.ds.current.Stage)
		}

		logx.MaybeLog(dispatch(t.dispatcher, dispatchTimeout, agentutil.LogEvent(t.local, err.Error())))
		return t.ds.current, dctx.Done(err)
	}

	d = agent.Deploy{Archive: &archive, Options: &opts, Stage: agent.Deploy_Deploying}
	t.update(dctx, d, agentutil.KeepOldestN(t.keepN))

	if err = writeDeployMetadata(dctx.Root, d); err != nil {
		logx.MaybeLog(dispatch(t.dispatcher, dispatchTimeout, agentutil.LogEvent(t.local, err.Error())))
		return d, dctx.Done(errors.WithStack(err))
	}

	logx.MaybeLog(dctx.Dispatch(agentutil.DeployEvent(dctx.Local, d)))

	if err = downloadArchive(t.dlreg, dctx); err != nil {
		return d, dctx.Done(err)
	}

	t.deployer.Deploy(dctx)

	return d, nil
}

// Cancel ...
func (t *Coordinator) Cancel() {
	t.m.Lock()
	defer t.m.Unlock()
	log.Println("cancelling deploy", *t.ds.state == coordinatorDeploying)
	if ok := atomic.CompareAndSwapUint32(t.ds.state, coordinatorDeploying, coordinaterWaiting); ok {
		t.ds.currentContext.Cancel(errors.New("deploy cancel signal received"))
		logx.MaybeLog(dispatch(t.dispatcher, dispatchTimeout, agentutil.LogEvent(t.local, "cancelled deploy")))
	} else {
		log.Println("ignored cancel not deploying", *t.ds.state == coordinatorDeploying)
	}
}

func (t *Coordinator) update(dctx DeployContext, d agent.Deploy, c agentutil.Cleaner) agent.Deploy {
	t.m.Lock()
	defer t.m.Unlock()

	t.ds.current = d
	t.ds.currentContext = dctx
	t.cleanup = c

	return d
}

func downloadArchive(dlreg storage.DownloadFactory, dctx DeployContext) error {
	dctx.Log.Printf("deploy recieved: deployID(%s) primary(%s) location(%s)\n", dctx.ID, dctx.Archive.Peer.Name, dctx.Archive.Location)
	defer dctx.Log.Printf("deploy complete: deployID(%s) primary(%s) location(%s)\n", dctx.ID, dctx.Archive.Peer.Name, dctx.Archive.Location)

	dctx.Log.Println("attempting to download", dctx.Archive.Location, dctx.ArchiveRoot)
	if err := archive.Unpack(dctx.ArchiveRoot, dlreg.New(dctx.Archive.Location).Download(dctx.deadline, dctx.Archive)); err != nil {
		return errors.Wrapf(err, "retrieve archive")
	}

	dctx.Log.Println("completed download", dctx.ArchiveRoot)
	return nil
}

// ResultBus bus for deploy results.
func ResultBus(in chan DeployResult, out ...chan DeployResult) {
	for result := range in {
		for _, dst := range out {
			dst <- result
		}
	}
}

func dispatch(d dispatcher, timeout time.Duration, m ...agent.Message) error {
	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()

	return d.Dispatch(ctx, m...)
}
