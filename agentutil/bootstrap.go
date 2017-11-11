package agentutil

import (
	"bytes"
	"log"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"bitbucket.org/jatone/bearded-wookie/agent"
)

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

// Bootstrap ...
func Bootstrap(local agent.Peer, c cluster, creds credentials.TransportCredentials) (err error) {
	var (
		status agent.Status
		latest agent.Archive
		client agent.Client
	)

	tcreds := grpc.WithTransportCredentials(creds)
	log.Println("--------------- bootstrap -------------")
	defer log.Println("--------------- bootstrap -------------")

	if latest, err = DetermineLatestArchive(c, tcreds); err != nil {
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

	if len(status.Deployments) > 0 && bytes.Compare(latest.DeploymentID, status.Deployments[0].DeploymentID) == 0 {
		log.Println("latest already deployed")
		return nil
	}

	if err = client.Deploy(latest); err != nil {
		return errors.Wrap(err, "failed to deploy latest")
	}

	return nil
}
