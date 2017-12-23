package deployment

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/x/logx"
)

// New Builds a deployment Coordinator.
func New(local agent.Peer, d deployer, options ...CoordinatorOption) Coordinator {
	const (
		defaultKeepN = 3
	)

	coord := &coordinator{
		local:        local,
		deployer:     d,
		Mutex:        &sync.Mutex{},
		latestDeploy: agent.Deploy{Stage: agent.Deploy_Completed},
		completed:    make(chan DeployResult),
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
	keepN        int // never set manually. always set by CoordinatorOptionKeepN
	root         string
	deploysRoot  string // never set manually. always set by CoordinatorOptionRoot
	local        agent.Peer
	deployer     deployer
	dispatcher   dispatcher
	cleanup      agentutil.Cleaner // never set manually. always set by CoordinatorOptionKeepN
	completed    chan DeployResult
	latestDeploy agent.Deploy
	*sync.Mutex
}

func (t *coordinator) background() {
	// by default keep the oldest deploys. if we have a successful deploy then keep the newest.
	for done := range t.completed {
		// cleanup workspace directory prior to execution. this leaves the last deployment
		// available until the next run.
		if soft := agentutil.MaybeClean(t.cleanup)(agentutil.Dirs(t.deploysRoot)); soft != nil {
			soft = errors.Wrap(soft, "failed to clear workspace directory")
			done.Dispatch(agentutil.LogEvent(done.Local, soft.Error()))
			log.Println(soft)
		}

		d := done.complete()
		switch d.Stage {
		case agent.Deploy_Completed:
			t.update(d, agentutil.KeepNewestN(t.keepN))
		default:
			t.update(d, agentutil.KeepOldestN(t.keepN))
		}
	}
}

// Info about the state of the agent.
func (t *coordinator) Deployments() (_psIgnored agent.Deploy, _ignored []*agent.Archive, err error) {
	var (
		archives []agent.Archive
	)

	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	if archives, err = readAllArchiveMetadata(t.deploysRoot); err != nil {
		return agent.Deploy{}, _ignored, err
	}

	sort.Slice(archives, func(i int, j int) bool {
		a, b := archives[i], archives[j]
		return a.Ts > b.Ts
	})

	return t.latestDeploy, archivePointers(archives...), nil
}

func (t *coordinator) Deploy(archive *agent.Archive) (err error) {
	var (
		d    agent.Deploy
		dctx DeployContext
	)

	if d = t.start(*archive); d.Stage != agent.Deploy_Deploying {
		return status(d.Stage)
	}

	if dctx, err = NewDeployContext(t.deploysRoot, t.local, *archive, DeployContextOptionCompleted(t.completed), DeployContextOptionDispatcher(t.dispatcher)); err != nil {
		return err
	}

	logx.MaybeLog(dctx.Dispatch(agentutil.DeployEvent(dctx.Local, d)))

	if err = writeArchiveMetadata(dctx); err != nil {
		return err
	}

	t.deployer.Deploy(dctx)

	return nil
}

func (t *coordinator) start(a agent.Archive) agent.Deploy {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	switch t.latestDeploy.Stage {
	case agent.Deploy_Completed, agent.Deploy_Failed:
	default:
		return agent.Deploy{Archive: &a, Stage: agent.Deploy_Failed}
	}

	t.latestDeploy = agent.Deploy{Archive: &a, Stage: agent.Deploy_Deploying}

	return t.latestDeploy
}

func (t *coordinator) update(d agent.Deploy, c agentutil.Cleaner) {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	t.latestDeploy = d
	t.cleanup = c
}
