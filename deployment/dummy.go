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

func (t dummy) Deploy(completed chan error) error {
	go func() {
		log.Println("deploying")
		defer log.Println("deploy complete")

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
