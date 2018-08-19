package bootstrap

import (
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
			log.Println("---------------------- bootstrap failed ----------------------\n", err)
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
		latest       agent.Deploy
		latestLocal  agent.Deploy
		latestQuorum agent.Deploy
		deploy       deployment.DeployResult
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
	if latestLocal, err = agentutil.LocalLatestDeployment(local, dialer); err != nil {
		return err
	}

	if latestQuorum, err = agentutil.QuorumLatestDeployment(c, dialer); err != nil {
		ignore := func(err error) error {
			cause := errors.Cause(err)

			switch cause {
			case agentutil.ErrActiveDeployment:
				// ignore active deployments when initialling bootstrapping,
				// we'll catch it at the end when we validate the version.
				return nil
			case agentutil.ErrNoDeployments:
				// ignore no deployment, we'll fallback to retrieving the latest
				// from the agents themselves.
				return nil
			}

			return err
		}

		if ignore(err) != nil {
			return err
		}
	}

	latest = latestQuorum
	if cause := errors.Cause(err); cause == agentutil.ErrNoDeployments || cause == agentutil.ErrActiveDeployment {
		log.Println(errors.Wrap(err, "no valid deployments available from quorum, fallback to latest from agents"))
		if latest, err = agentutil.DetermineLatestDeployment(c, dialer); err != nil {
			switch cause := errors.Cause(err); cause {
			case agentutil.ErrNoDeployments:
				log.Println(cause)
				return nil
			default:
				return errors.Wrap(cause, "failed to determine latest archive to bootstrapping")
			}
		}
	}

	if agentutil.SameArchive(latestQuorum.Archive, latestLocal.Archive) {
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
	case deploy = <-deployResults:
	}

	if err = deploy.Error; err != nil {
		return errors.Wrap(err, "deployment failed")
	}

	// again retrieve the latest deployment information from the cluster.
	// if a deploy is ongoing or is different from the deploy we just used to bootstrap
	// we want to consider the bootstrap a failure and retry.
	if latestQuorum, err = agentutil.QuorumLatestDeployment(c, dialer); err != nil {
		return errors.Wrap(err, "failed to determine latest deployment from quorum, retrying")
	}

	if agentutil.SameArchive(latestQuorum.Archive, &deploy.Archive) {
		return errors.WithStack(agentutil.ErrDifferentDeployment)
	}

	return nil
}
