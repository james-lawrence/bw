package agentutil

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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
func BootstrapUntilSuccess(ctx context.Context, local agent.Peer, c cluster, creds credentials.TransportCredentials, d deployer) bool {
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

		if err = Bootstrap(local, c, creds, d); err != nil {
			log.Println("failed bootstrap", err)
			time.Sleep(bs.Backoff(i))
			log.Println("REATTEMPT BOOTSTRAP")
			continue
		}

		return true
	}
}

// Bootstrap ...
func Bootstrap(local agent.Peer, c cluster, creds credentials.TransportCredentials, d deployer) (err error) {
	var (
		status agent.StatusResponse
		latest agent.Deploy
		client agent.Client
	)

	tcreds := grpc.WithTransportCredentials(creds)
	log.Println("--------------- bootstrap -------------")
	defer log.Println("--------------- bootstrap -------------")

	if latest, err = DetermineLatestDeployment(c, tcreds); err != nil {
		switch cause := errors.Cause(err); cause {
		case ErrNoDeployments:
			log.Println("no deployments found")
			return nil
		default:
			return errors.Wrap(cause, "failed to determine latest archive to bootstrapping")
		}
	}

	if client, err = DialPeer(local, tcreds); err != nil {
		return errors.Wrap(err, "failed to connect to local server")
	}

	if status, err = client.Info(); err != nil {
		return errors.Wrap(err, "failed to retrieve local")
	}

	if err = client.Close(); err != nil {
		return errors.WithStack(err)
	}

	if len(status.Deployments) > 0 && bytes.Compare(latest.Archive.DeploymentID, status.Deployments[0].Archive.DeploymentID) == 0 {
		log.Println("latest already deployed")
		return nil
	}

	if _, err = d.Deploy(agent.DeployOptions{}, *latest.Archive); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
