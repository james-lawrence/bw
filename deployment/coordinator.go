package deployment

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

// New Builds a deployment Coordinator.
func New(d deployer, options ...CoordinatorOption) Coordinator {
	coord := &deployment{
		keepN:     3,
		deployer:  d,
		Mutex:     &sync.Mutex{},
		status:    ready{},
		completed: make(chan error),
	}

	// default options.
	CoordinatorOptionRoot(os.TempDir())(coord)

	for _, opt := range options {
		opt(coord)
	}

	go coord.init()

	return coord
}

// CoordinatorOption options for the deployment coordinator.
type CoordinatorOption func(*deployment)

// CoordinatorOptionRoot - root directory for performing deployments.
func CoordinatorOptionRoot(root string) CoordinatorOption {
	return func(d *deployment) {
		d.root = root
		d.deploysRoot = filepath.Join(root, "deployments")
	}
}

// CoordinatorOptionKeepN the number of previous deploys to keep.
func CoordinatorOptionKeepN(n int) CoordinatorOption {
	return func(d *deployment) {
		d.keepN = n
	}
}

type deployment struct {
	keepN       int
	root        string
	deploysRoot string // never set manually. always set by CoordinatorOptionRoot
	deployer
	*sync.Mutex
	status    Status
	completed chan error
}

func (t *deployment) init() {
	// TODO change completed to be a channel of DeployResult, which is a DeployContext, and an error.
	for err := range t.completed {
		// by default keep the oldest deploys. if we have a successful deploy then keep the newest.
		cleanup := agentutil.KeepOldestN(t.keepN)

		if err != nil {
			log.Printf("deployment failed: %+v\n", err)
			t.update(failed{})
		} else {
			t.update(ready{})
			cleanup = agentutil.KeepNewestN(t.keepN)
		}

		// cleanup workspace directory.
		if soft := agentutil.MaybeClean(cleanup)(agentutil.Dirs(t.deploysRoot)); soft != nil {
			log.Println("failed to clean workspace directory", soft)
		}
	}
}

// Info about the state of the agent.
func (t *deployment) Info() (_ignored agent.AgentInfo, err error) {
	var (
		archives []agent.Archive
	)

	t.Mutex.Lock()
	defer t.Mutex.Unlock()

	if archives, err = readAllArchiveMetadata(t.deploysRoot); err != nil {
		return _ignored, err
	}

	sort.Slice(archives, func(i int, j int) bool {
		a, b := archives[i], archives[j]
		return a.Ts > b.Ts
	})

	return agent.AgentInfo{
		Status:      AgentStateFromStatus(t.status),
		Deployments: archivePointers(archives...),
	}, nil
}

func (t *deployment) Deploy(archive *agent.Archive) (err error) {
	var (
		dctx DeployContext
	)

	if err := t.startDeploy(); err != nil {
		return err
	}

	if dctx, err = NewDeployContext(t.deploysRoot, *archive); err != nil {
		return err
	}

	if err = writeArchiveMetadata(dctx); err != nil {
		return err
	}

	return t.deployer.Deploy(dctx, t.completed)
}

func (t *deployment) startDeploy() Status {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	if !(IsReady(t.status) || IsFailed(t.status)) {
		return t.status
	}
	t.status = deploying{}
	return nil
}

func (t *deployment) update(s Status) {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.status = s
}
