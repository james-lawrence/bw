package bootstrap

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
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

type option func(*UntilSuccess2)

// OptionMaxAttempts set maximum number of attempts.
func OptionMaxAttempts(n int) func(*UntilSuccess2) {
	return func(us *UntilSuccess2) {
		us.maxAttempts = n
	}
}

// OptionBackoff set strategy for backing off.
func OptionBackoff(bs backoff.Strategy) func(*UntilSuccess2) {
	return func(us *UntilSuccess2) {
		us.bs = bs
	}
}

// NewUntilSuccess continuously bootstraps until it succeeds or hits maximum attempts.
func NewUntilSuccess(options ...option) UntilSuccess2 {
	us := UntilSuccess2{
		maxAttempts: math.MaxInt64, // effectively forever.
		bs:          backoff.Maximum(time.Minute, backoff.Exponential(2*time.Second)),
	}

	for _, opt := range options {
		opt(&us)
	}

	return us
}

// UntilSuccess2 attempts to bootstrap until max attempts or success.
type UntilSuccess2 struct {
	maxAttempts int
	bs          backoff.Strategy
}

// Run bootstrapping process until it succeeds
func (t UntilSuccess2) Run(local agent.Peer, c cluster, dialer dialer, coord deployment.Coordinator) bool {
	for i := 0; i < t.maxAttempts; i++ {
		if err := Bootstrap(local, c, dialer, coord); err != nil {
			log.Println(errors.Wrap(err, "bootstrap attempt failed"))
			time.Sleep(t.bs.Backoff(i))
			continue
		}

		log.Println("--------------- bootstrap complete ---------------")
		return true
	}

	log.Println("--------------- bootstrap failure ---------------")
	return false
}

// Bootstrap a server with the latest deploy.
func Bootstrap(local agent.Peer, c cluster, dialer dialer, coord deployment.Coordinator) (err error) {
	ignore := func(err error) error {
		cause := errors.Cause(err)

		switch cause {
		case agentutil.ErrActiveDeployment:
			// ignore active deployments when initially bootstrapping,
			// we'll catch it at the end when we validate the version.
			return nil
		case agentutil.ErrNoDeployments:
			// ignore no deployment, we'll fallback to retrieving the latest
			// from the agents themselves.
			return nil
		}

		return err
	}

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

	log.Println("--------------- bootstrap attempt initiated -------------")
	defer log.Println("--------------- bootstrap attempt completed -------------")
	if latestLocal, err = agentutil.LocalLatestDeployment(local, dialer); ignore(err) != nil {
		return errors.Wrap(err, "latest local failed")
	}

	if latestQuorum, err = agentutil.QuorumLatestDeployment(c, dialer); ignore(err) != nil {
		return errors.Wrap(err, "latest quorum failed")
	}

	if agentutil.IsActiveDeployment(err) && agentutil.SameArchive(latestQuorum.Archive, latestLocal.Archive) {
		return errors.Wrap(err, "active deploy matches the local deployment, waiting for deployment to complete")
	}

	if cause := errors.Cause(err); cause == agentutil.ErrNoDeployments {
		log.Println(errors.Wrap(err, "no valid deployments available from quorum, fallback to latest from agents"))
		if latestQuorum, err = agentutil.DetermineLatestDeployment(c, dialer); err != nil {
			switch cause := errors.Cause(err); cause {
			case agentutil.ErrNoDeployments:
				log.Println(errors.Wrap(cause, "latest deployment discovery found no deployments"))
				return nil
			default:
				return errors.Wrap(cause, "failed to determine latest archive to bootstrap")
			}
		}
	}

	latest = latestQuorum

	if agentutil.SameArchive(latest.Archive, latestLocal.Archive) {
		log.Println("latest already deployed -", spew.Sdump(latestLocal))
		return nil
	}

	opts := *latest.Options

	deadline, cancel := context.WithTimeout(context.Background(), time.Duration(opts.Timeout))
	defer cancel()

	log.Println("bootstrapping", bw.RandomID(latest.Archive.DeploymentID), "with options", spew.Sdump(opts))
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
		return errors.Wrap(err, "failed to determine latest deployment from quorum, retrying 2")
	}

	if !agentutil.SameArchive(latestQuorum.Archive, &deploy.Archive) {
		return errors.WithStack(agentutil.ErrDifferentDeployment)
	}

	return nil
}
