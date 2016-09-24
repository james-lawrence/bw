package deployment

import (
	"crypto/md5"
	"encoding/json"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"bitbucket.org/jatone/bearded-wookie/packagekit"
)

type packageByID []packagekit.Package

// Methods required by sort Interface
func (t packageByID) Len() int           { return len(t) }
func (t packageByID) Less(i, j int) bool { return t[i].ID < t[j].ID }
func (t packageByID) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

const sleepy = 60

func NewDefaultCoordinator() (Coordinator, error) {
	pkgkit, err := packagekit.NewClient()
	if err != nil {
		return nil, err
	}

	signal := make(chan struct{})
	completed := make(chan struct{})
	coord := New(pkgkit, signal, completed).(*deployment)
	go func() {
		for _ = range signal {
			log.Println("deploying")
			defer log.Println("deploy complete")

			select {
			case _ = <-time.After(time.Duration(rand.Intn(sleepy)) * time.Second):
				coord.completed <- struct{}{}
			case _ = <-time.After(time.Duration(rand.Intn(sleepy)*2) * time.Second):
				coord.failed <- struct{}{}
			}
		}
	}()

	return coord, nil
}

// New Builds a deployment Coordinator.
// Pass in a packagekit.Client implementation
func New(client packagekit.Client, signal, completed chan struct{}) Coordinator {
	coord := &deployment{
		pkgkit:    client,
		Mutex:     &sync.Mutex{},
		status:    ready{},
		signal:    signal,
		completed: completed,
		failed:    make(chan struct{}),
	}

	go coord.init()
	return coord
}

type deployment struct {
	pkgkit packagekit.Client
	*sync.Mutex
	status    Status
	signal    chan struct{}
	completed chan struct{}
	failed    chan struct{}
}

func (t *deployment) init() {
	for {
		select {
		case _ = <-t.completed:
			t.update(ready{})
		case _ = <-t.failed:
			t.update(failed{})
		}
	}
}

// Status of the deployment Coordinator
func (t *deployment) Status() error {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.status
}

func (t *deployment) Deploy() error {
	if err := t.startDeploy(); err != nil {
		return err
	}

	// signal the deploy
	t.signal <- struct{}{}

	return nil
}

func (t *deployment) startDeploy() Status {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	if !IsReady(t.status) {
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

func (t *deployment) SystemStateChecksum() ([]byte, error) {
	var (
		hasher = md5.New()
	)

	packages, err := t.Packages()
	if err != nil {
		return nil, err
	}

	// sort to guarentee the ordering of the pages.
	sort.Sort(packageByID(packages))

	json.NewEncoder(hasher).Encode(packages)

	return hasher.Sum(nil), nil
}

func (t *deployment) Packages() ([]packagekit.Package, error) {
	var (
		err error
		tx  packagekit.Transaction
	)

	if tx, err = t.pkgkit.CreateTransaction(); err != nil {
		return nil, err
	}

	return tx.Packages(packagekit.FilterInstalled)
}
