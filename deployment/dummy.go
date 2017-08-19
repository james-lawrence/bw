package deployment

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"time"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

// NewDummyCoordinator Builds a coordinator that uses a fake deployer.
func NewDummyCoordinator() Coordinator {
	const sleepy = 60
	return New(dummy{
		sleepy: sleepy,
	})
}

type dummy struct {
	sleepy int
}

func (t dummy) Deploy(archive *agent.Archive, completed chan error) error {
	go func() {
		log.Printf("deploy recieved: deployID(%s) leader(%s) location(%s)\n", hex.EncodeToString(archive.DeploymentID), archive.Leader, archive.Location)
		defer log.Printf("deploy complete: deployID(%s) leader(%s) location(%s)\n", hex.EncodeToString(archive.DeploymentID), archive.Leader, archive.Location)

		completedDuration := time.Duration(rand.Intn(t.sleepy)) * time.Second
		failedDuration := time.Duration(rand.Intn(t.sleepy)*2) * time.Second
		select {
		case _ = <-time.After(completedDuration):
			completed <- nil
		case _ = <-time.After(failedDuration):
			log.Println("failed deployment due to timeout", failedDuration)
			completed <- timeout{Duration: failedDuration}
		}
	}()

	return nil
}

type timeout struct {
	time.Duration
}

func (t timeout) Error() string {
	return fmt.Sprintf("timed out after: %s", t.Duration)
}

func (t timeout) Timeout() {}
