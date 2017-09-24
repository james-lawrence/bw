package deployment

import (
	"fmt"
	"log"
	"math/rand"
	"time"
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

func (t dummy) Deploy(dctx DeployContext) error {
	go func() {
		log.Printf("deploy recieved: deployID(%s) leader(%s) location(%s)\n", dctx.ID, dctx.Archive.Peer.Name, dctx.Archive.Location)
		defer log.Printf("deploy complete: deployID(%s) leader(%s) location(%s)\n", dctx.ID, dctx.Archive.Peer.Name, dctx.Archive.Location)

		completedDuration := time.Duration(rand.Intn(t.sleepy)) * time.Second
		failedDuration := time.Duration(rand.Intn(t.sleepy)*2) * time.Second
		select {
		case _ = <-time.After(completedDuration):
			dctx.Done(nil)
		case _ = <-time.After(failedDuration):
			log.Println("failed deployment due to timeout", failedDuration)
			dctx.Done(timeout{Duration: failedDuration})
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
