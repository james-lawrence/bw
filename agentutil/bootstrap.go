package agentutil

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/backoff"
)

// Coordinator is in charge of coordinating deployments.
type deployer interface {
	// Deploy trigger a deploy
	Deploy(agent.DeployOptions, agent.Archive) (agent.Deploy, error)
	Deployments() ([]agent.Deploy, error)
}

type errString string

func (t errString) Error() string {
	return string(t)
}

const (
	// ErrNoDeployments ...
	ErrNoDeployments = errString("no deployments found")
	// ErrFailedDeploymentQuorum ...
	ErrFailedDeploymentQuorum = errString("unable to achieve latest deployment quorum")
)

// BootstrapUntilSuccess continuously bootstraps until it succeeds.
func BootstrapUntilSuccess(ctx context.Context, local agent.Peer, c cluster, dialer agent.Dialer, d deployer) bool {
	var (
		err error
	)

	bs := backoff.Maximum(10*time.Second, backoff.Exponential(2*time.Second))

	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if err = Bootstrap(local, c, dialer, d); err != nil {
			log.Println("failed bootstrap", err)
			time.Sleep(bs.Backoff(i))
			log.Println("REATTEMPT BOOTSTRAP")
			continue
		}

		return true
	}
}

// Bootstrap ...
func Bootstrap(local agent.Peer, c cluster, dialer agent.Dialer, d deployer) (err error) {
	var (
		status agent.StatusResponse
		latest agent.Deploy
		client agent.Client
	)

	log.Println("--------------- bootstrap -------------")
	defer log.Println("--------------- bootstrap -------------")

	if latest, err = DetermineLatestDeployment(c, dialer); err != nil {
		switch cause := errors.Cause(err); cause {
		case ErrNoDeployments:
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
		log.Println("latest already deployed", spew.Sdump(status))
		return nil
	}

	log.Println("bootstrapping with", spew.Sdump(latest))
	if _, err = d.Deploy(agent.DeployOptions{}, *latest.Archive); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
