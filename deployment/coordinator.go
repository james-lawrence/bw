package deployment

import (
	"log"
	"sync"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

// New Builds a deployment Coordinator.
func New(d deployer) Coordinator {
	coord := &deployment{
		deployer:  d,
		Mutex:     &sync.Mutex{},
		status:    ready{},
		completed: make(chan error),
	}

	go coord.init()
	return coord
}

type deployment struct {
	deployer
	*sync.Mutex
	status    Status
	completed chan error
}

func (t *deployment) init() {
	for err := range t.completed {
		if err != nil {
			log.Printf("deployment failed: %+v\n", err)
			t.update(failed{})
		} else {
			t.update(ready{})
		}

	}
}

// Status of the deployment Coordinator
func (t *deployment) Status() error {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.status
}

func (t *deployment) Deploy(archive *agent.Archive) error {
	if err := t.startDeploy(); err != nil {
		return err
	}

	return t.deployer.Deploy(archive, t.completed)
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
