package bootstrap

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/deployment"
)

type dialer interface {
	Dial(agent.Peer) (zeroc agent.Client, err error)
}

type cluster interface {
	Local() agent.Peer
	Peers() []agent.Peer
	Quorum() []agent.Peer
	Connect() agent.ConnectResponse
}

// UntilSuccess continuously bootstraps until it succeeds or hits maximum attempts.
func UntilSuccess(maxAttempts int, local agent.Peer, c cluster, dialer dialer, coord deployment.Coordinator) bool {
	var (
		err error
	)

	bs := backoff.Maximum(10*time.Second, backoff.Exponential(2*time.Second))

	for i := 0; i < maxAttempts; i++ {
		if err = Bootstrap(local, c, dialer, coord); err != nil {
			log.Println("failed bootstrap", err)
			time.Sleep(bs.Backoff(i))
			log.Println("REATTEMPT BOOTSTRAP")
			continue
		}

		return true
	}

	return false
}

// Bootstrap a server with the latest deploy.
func Bootstrap(local agent.Peer, c cluster, dialer dialer, coord deployment.Coordinator) (err error) {
	var (
		status agent.StatusResponse
		latest agent.Deploy
		client agent.Client
	)

	// Here we clone the coordinator to override some behaviours around dispatching and observations.
	deployResults := make(chan deployment.DeployResult)
	coord = deployment.CloneCoordinator(
		coord,
		deployment.CoordinatorOptionDispatcher(agentutil.LogDispatcher{}),
		deployment.CoordinatorOptionDeployResults(deployResults),
	)

	log.Println("--------------- bootstrap -------------")
	defer log.Println("--------------- bootstrap -------------")

	if latest, err = agentutil.DetermineLatestDeployment(c, dialer); err != nil {
		switch cause := errors.Cause(err); cause {
		case agentutil.ErrNoDeployments:
			log.Println(cause)
			return nil
		default:
			return errors.Wrap(cause, "failed to determine latest archive to bootstrapping")
		}
	}

	if client, err = dialer.Dial(local); err != nil {
		return errors.Wrap(err, "failed to connect to local server")
	}

	if status, err = client.Info(); err != nil {
		return errors.Wrap(err, "failed to retrieve local")
	}

	if err = client.Close(); err != nil {
		return errors.WithStack(err)
	}

	if len(status.Deployments) > 0 && status.Deployments[0].Stage == agent.Deploy_Completed && bytes.Compare(latest.Archive.DeploymentID, status.Deployments[0].Archive.DeploymentID) == 0 {
		log.Println("latest already deployed")
		return nil
	}

	opts := *latest.Options

	deadline, cancel := context.WithTimeout(context.Background(), time.Duration(opts.Timeout))
	defer cancel()
	log.Println("bootstrapping with options", spew.Sdump(opts))
	if _, err = coord.Deploy(opts, *latest.Archive); err != nil {
		return errors.WithStack(err)
	}

	select {
	case <-deadline.Done():
		return errors.Wrap(deadline.Err(), "failed to bootstrap timeout")
	case deploy := <-deployResults:
		return errors.Wrap(deploy.Error, "deployment failed")
	}
}
