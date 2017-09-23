package deployment

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"bitbucket.org/jatone/bearded-wookie/agentutil"
	gagent "bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

// New Builds a deployment Coordinator.
func New(d deployer, options ...CoordinatorOption) Coordinator {
	const (
		defaultKeepN = 3
	)

	coord := &deployment{
		deployer:  d,
		Mutex:     &sync.Mutex{},
		status:    ready{},
		completed: make(chan DeployResult),
	}

	// default options.
	CoordinatorOptionRoot(os.TempDir())(coord)
	CoordinatorOptionKeepN(defaultKeepN)

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
		d.deploysRoot = filepath.Join(root, "deployments")
	}
}

// CoordinatorOptionKeepN the number of previous deploys to keep.
func CoordinatorOptionKeepN(n int) CoordinatorOption {
	return func(d *deployment) {
		d.keepN = n
		d.cleanup = agentutil.KeepOldestN(n)
	}
}

type deployment struct {
	keepN       int // never set manually. always set by CoordinatorOptionKeepN
	root        string
	deploysRoot string // never set manually. always set by CoordinatorOptionRoot
	deployer
	cleanup agentutil.Cleaner // never set manually. always set by CoordinatorOptionKeepN
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
			log.Println("failed to clean workspace directory", soft)
		}

		if done.Error != nil {
			log.Printf("deployment failed: %+v\n", done.Error)
			t.update(failed{}, agentutil.KeepOldestN(t.keepN))
			continue
		}

		t.update(ready{}, agentutil.KeepNewestN(t.keepN))
	}
}

// Info about the state of the agent.
func (t *deployment) Info() (_ignored gagent.AgentInfo, err error) {
	var (
		archives []gagent.Archive
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

	return gagent.AgentInfo{
		Status:      AgentStateFromStatus(t.status),
		Deployments: archivePointers(archives...),
	}, nil
}

func (t *deployment) Deploy(archive *gagent.Archive) (err error) {
	var (
		dctx DeployContext
	)

	if err := t.start(); err != nil {
		return err
	}

	if dctx, err = NewDeployContext(t.deploysRoot, *archive, DeployContextOptionCompleted(t.completed)); err != nil {
		return err
	}

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
