package agentutil

import (
	"log"

	"bitbucket.org/jatone/bearded-wookie/clustering/raftutil"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

// NewBootstrapper generates a new bootstrapper for the given server.
func NewBootstrapper(c cluster, creds credentials.TransportCredentials) Bootstrapper {
	buffer := make(chan raftutil.Event, 100)

	b := Bootstrapper{
		In:    buffer,
		c:     c,
		creds: grpc.WithTransportCredentials(creds),
	}

	go b.background()

	return b
}

// Bootstrapper monitors the raft cluster for new nodes and bootstraps them.
type Bootstrapper struct {
	c     cluster
	creds grpc.DialOption
	In    chan raftutil.Event
}

// Observer implements raftutil.clusterObserver
func (t Bootstrapper) Observer(e raftutil.Event) {
	t.In <- e
}

// Start ...
func (t Bootstrapper) background() {
	var (
		err    error
		peer   agent.Peer
		latest agent.Archive
		c      agent.Client
	)

	for o := range t.In {
		if o.Type != raftutil.EventJoined {
			log.Println("ignoring bootstrap event, peer did not join", o.Peer.Name, o.Peer.Address())
			continue
		}

		if peer, err = NodeToPeer(o.Peer); err != nil {
			log.Println("failed to convert node to peer", err)
			continue
		}

		log.Println("--------------- bootstrap observed -------------", o)

		if latest, err = DetermineLatestArchive(t.c, t.creds); err != nil {
			log.Println("failed to determine latest archive prior to bootstrapping", err)
			continue
		}

		if c, err = DialPeer(peer, t.creds); err != nil {
			log.Println("failed to connect to new peer", o.Peer.String(), c)
			continue
		}

		if err = c.Deploy(latest); err != nil {
			log.Println("failed to deploy to new peer", o.Peer.String(), err)
			continue
		}
	}
}
