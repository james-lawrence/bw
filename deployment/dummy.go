package deployment

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/errorsx"
)

// NewDummyCoordinator Builds a coordinator that uses a fake deployer.
func NewDummyCoordinator(p *agent.Peer) Coordinator {
	const sleepy = 60
	return New(p, dummy{sleepy: sleepy})
}

type dummy struct {
	sleepy int
}

func (t dummy) Deploy(dctx *DeployContext) {
	go func() {
		log.Printf("deploy recieved: deployID(%s) leader(%s) location(%s)\n", dctx.ID, dctx.Archive.Peer.Name, dctx.Archive.Location)
		defer log.Printf("deploy complete: deployID(%s) leader(%s) location(%s)\n", dctx.ID, dctx.Archive.Peer.Name, dctx.Archive.Location)

		completedDuration := time.Duration(rand.Intn(t.sleepy)) * time.Second
		failedDuration := time.Duration(rand.Intn(t.sleepy)*2) * time.Second
		select {
		case <-time.After(completedDuration):
			errorsx.Log(dctx.Done(nil))
		case <-time.After(failedDuration):
			log.Println("failed deployment due to timeout", failedDuration)
			errorsx.Log(dctx.Done(timeout{Duration: failedDuration}))
		}
	}()
}

type timeout struct {
	time.Duration
}

func (t timeout) Error() string {
	return fmt.Sprintf("timed out after: %s", t.Duration)
}

func (t timeout) Timeout() {}
