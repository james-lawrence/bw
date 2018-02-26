package deployment

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
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

	coord := &coordinator{
		local:     local,
		deployer:  d,
		Mutex:     &sync.Mutex{},
		completed: make(chan DeployResult),
	}

	// default options.
	CoordinatorOptionRoot(os.TempDir())(coord)
	CoordinatorOptionKeepN(defaultKeepN)(coord)
	CoordinatorOptionDispatcher(logDispatcher{})(coord)

	for _, opt := range options {
		opt(coord)
	}

	go coord.background()

	return coord
}

// CoordinatorOption options for the deployment coordinator.
type CoordinatorOption func(*coordinator)

// CoordinatorOptionRoot - root directory for performing deployments.
func CoordinatorOptionRoot(root string) CoordinatorOption {
	return func(d *coordinator) {
		d.root = root
		d.deploysRoot = filepath.Join(root, bw.DirDeploys)
	}
}

// CoordinatorOptionKeepN the number of previous deploys to keep.
func CoordinatorOptionKeepN(n int) CoordinatorOption {
	return func(d *coordinator) {
		d.keepN = n
		d.cleanup = agentutil.KeepOldestN(n)
	}
}

// CoordinatorOptionDispatcher sets the dispatcher for the coordinator.
func CoordinatorOptionDispatcher(di dispatcher) CoordinatorOption {
	return func(d *coordinator) {
		d.dispatcher = di
	}
}

type coordinator struct {
	keepN         int // never set manually. always set by CoordinatorOptionKeepN
	root          string
	deploysRoot   string // never set manually. always set by CoordinatorOptionRoot
	local         agent.Peer
	deployer      deployer
	dispatcher    dispatcher
	cleanup       agentutil.Cleaner // never set manually. always set by CoordinatorOptionKeepN
	completed     chan DeployResult
	currentDeploy agent.Deploy
	deploying     uint32
	*sync.Mutex
}

func (t *coordinator) background() {
	// by default keep the oldest deploys. if we have a successful deploy then keep the newest.
	for done := range t.completed {
		d := done.complete()

		if err := writeDeployMetadata(done.Root, d); err != nil {
			log.Println("failed to write deploy metadata", err)
		}

		switch d.Stage {
		case agent.Deploy_Completed:
			t.update(d, agentutil.KeepNewestN(t.keepN))
		default:
			t.update(d, agentutil.KeepOldestN(t.keepN))
		}

		atomic.SwapUint32(&t.deploying, coordinaterWaiting)
	}
}

// Info about the state of the agent.
func (t *coordinator) Deployments() (deployments []agent.Deploy, err error) {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	if deployments, err = readAllDeployMetadata(t.deploysRoot); err != nil {
		return deployments, err
	}

	sort.Slice(deployments, func(i int, j int) bool {
		a, b := deployments[i], deployments[j]
		return a.Archive.Ts > b.Archive.Ts
	})

	return deployments, nil
}

func (t *coordinator) Deploy(opts agent.DeployOptions, archive agent.Archive) (d agent.Deploy, err error) {
	var (
		ok   bool
		dctx DeployContext
	)

	d = agent.Deploy{Archive: &archive, Stage: agent.Deploy_Deploying}

	// cleanup workspace directory prior to deployment. this leaves the last deployment
	// is available until the next run for debugging.
	// IMPORTANT: torrent storage relies on this behaviour in order to prevent
	// downloads from becoming permanently blocked waiting for the archive to be downloaded.
	// without this behaviour the torrent can be removed while nodes are still trying to deploy.
	// preventing further deploys.
	if soft := agentutil.MaybeClean(t.cleanup)(agentutil.Dirs(t.deploysRoot)); soft != nil {
		soft = errors.Wrap(soft, "failed to clear workspace directory")
		t.dispatcher.Dispatch(agentutil.LogEvent(t.local, soft.Error()))
		log.Println(soft)
	}

	if dctx, err = NewDeployContext(t.deploysRoot, t.local, archive, DeployContextOptionCompleted(t.completed), DeployContextOptionDispatcher(t.dispatcher)); err != nil {
		t.dispatcher.Dispatch(agentutil.LogEvent(t.local, err.Error()))
		return d, err
	}

	if err = writeDeployMetadata(dctx.Root, d); err != nil {
		t.dispatcher.Dispatch(agentutil.LogEvent(t.local, err.Error()))
		return d, errors.WithStack(err)
	}

	if ok = atomic.CompareAndSwapUint32(&t.deploying, coordinaterWaiting, coordinatorDeploying); !ok {
		err = errors.Errorf("already deploying: %s - %s", bw.RandomID(t.currentDeploy.Archive.DeploymentID).String(), t.currentDeploy.Stage)
		t.dispatcher.Dispatch(agentutil.LogEvent(t.local, err.Error()))
		return d, err
	}

	logx.MaybeLog(dctx.Dispatch(agentutil.DeployEvent(dctx.Local, d)))

	t.deployer.Deploy(dctx)

	return d, nil
}

func (t *coordinator) update(d agent.Deploy, c agentutil.Cleaner) agent.Deploy {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	t.currentDeploy = d
	t.cleanup = c

	return d
}
