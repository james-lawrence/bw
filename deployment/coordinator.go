package deployment

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/x/logx"
)

// New Builds a deployment Coordinator.
func New(local agent.Peer, d deployer, options ...CoordinatorOption) Coordinator {
	const (
		defaultKeepN = 3
	)

	coord := &deployment{
		local:     local,
		deployer:  d,
		Mutex:     &sync.Mutex{},
		status:    ready{},
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
type CoordinatorOption func(*deployment)

// CoordinatorOptionRoot - root directory for performing deployments.
func CoordinatorOptionRoot(root string) CoordinatorOption {
	return func(d *deployment) {
		d.root = root
		d.deploysRoot = filepath.Join(root, bw.DirDeploys)
	}
}

// CoordinatorOptionKeepN the number of previous deploys to keep.
func CoordinatorOptionKeepN(n int) CoordinatorOption {
	return func(d *deployment) {
		d.keepN = n
		d.cleanup = agentutil.KeepOldestN(n)
	}
}

// CoordinatorOptionDispatcher sets the dispatcher for the coordinator.
func CoordinatorOptionDispatcher(di dispatcher) CoordinatorOption {
	return func(d *deployment) {
		d.dispatcher = di
	}
}

type deployment struct {
	keepN       int // never set manually. always set by CoordinatorOptionKeepN
	root        string
	deploysRoot string // never set manually. always set by CoordinatorOptionRoot
	local       agent.Peer
	deployer    deployer
	dispatcher  dispatcher
	cleanup     agentutil.Cleaner // never set manually. always set by CoordinatorOptionKeepN
	*sync.Mutex
	status    Status
	completed chan DeployResult
}

func (t *deployment) background() {
	// by default keep the oldest deploys. if we have a successful deploy then keep the newest.
	for done := range t.completed {
		// cleanup workspace directory prior to execution. this leaves the last deployment
		// available until the next run.
		if soft := agentutil.MaybeClean(t.cleanup)(agentutil.Dirs(t.deploysRoot)); soft != nil {
			soft = errors.Wrap(soft, "failed to clear workspace directory")
			done.Dispatch(agentutil.LogEvent(done.Local, soft.Error()))
			log.Println(soft)
		}

		if done.Error != nil {
			t.update(failed{}, agentutil.KeepOldestN(t.keepN))
			done.deployFailed(done.Error)
			continue
		}

		done.deployComplete()
		t.update(ready{}, agentutil.KeepNewestN(t.keepN))
	}
}

// Info about the state of the agent.
func (t *deployment) Deployments() (_psIgnored agent.Peer_State, _ignored []*agent.Archive, err error) {
	var (
		archives []agent.Archive
	)

	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	if archives, err = readAllArchiveMetadata(t.deploysRoot); err != nil {
		return agent.Peer_Unknown, _ignored, err
	}

	sort.Slice(archives, func(i int, j int) bool {
		a, b := archives[i], archives[j]
		return a.Ts > b.Ts
	})

	return AgentStateFromStatus(t.status), archivePointers(archives...), nil
}

func (t *deployment) Deploy(archive *agent.Archive) (err error) {
	var (
		dctx DeployContext
	)

	if err := t.start(); err != nil {
		return err
	}

	if dctx, err = NewDeployContext(t.deploysRoot, t.local, *archive, DeployContextOptionCompleted(t.completed), DeployContextOptionDispatcher(t.dispatcher)); err != nil {
		return err
	}

	logx.MaybeLog(dctx.Dispatch(agentutil.DeployInitiatedEvent(dctx.Local, dctx.Archive)))

	if err = writeArchiveMetadata(dctx); err != nil {
		return err
	}

	return t.deployer.Deploy(dctx)
}

func (t *deployment) start() Status {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	if !(IsReady(t.status) || IsFailed(t.status)) {
		return t.status
	}

	t.status = deploying{}

	return nil
}

func (t *deployment) update(s Status, c agentutil.Cleaner) {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	t.status = s
	t.cleanup = c
}
