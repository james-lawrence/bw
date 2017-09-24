package agent

import (
	"log"
	"net"

	"bitbucket.org/jatone/bearded-wookie/clustering/raftutil"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
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
func NewBootstrapper(s Server) Bootstrapper {
	buffer := make(chan raftutil.Event, 100)

	b := Bootstrapper{
		In:     buffer,
		Server: s,
	}

	go b.background()

	return b
}

// Bootstrapper monitors the raft cluster for new nodes and bootstraps them.
type Bootstrapper struct {
	Server
	In chan raftutil.Event
}

// Observer implements raftutil.clusterObserver
func (t Bootstrapper) Observer(e raftutil.Event) {
	t.In <- e
}

// Start ...
func (t Bootstrapper) background() {
	var (
		err    error
		latest agent.Archive
		port   string
		c      Client
	)

	for o := range t.In {
		if o.Type != raftutil.EventJoined {
			log.Println("ignoring bootstrap event, node did not have join type")
			continue
		}

		log.Println("--------------- bootstrap observed -------------", o)
		if _, port, err = net.SplitHostPort(t.Server.Address.String()); err != nil {
			log.Println("failed to determine rpc port", err)
			continue
		}

		if latest, err = DetermineLatestArchive(t.Server.Address, t.Server.cluster, grpc.WithTransportCredentials(t.Server.creds)); err != nil {
			log.Println("failed to determine latest archive prior to bootstrapping", err)
			continue
		}

		if c, err = DialClient(net.JoinHostPort(o.Peer.Addr.String(), port), grpc.WithTransportCredentials(t.Server.creds)); err != nil {
			log.Println("failed to connect to new peer", o.Peer.String(), c)
			continue
		}

		if err = c.Deploy(latest); err != nil {
			log.Println("failed to deploy to new peer", o.Peer.String(), err)
			continue
		}
	}
}

// DetermineLatestArchive ...
func DetermineLatestArchive(addr net.Addr, c cluster, DialOptions ...grpc.DialOption) (latest agent.Archive, err error) {
	type result struct {
		a     *agent.Archive
		count int
	}

	var (
		quorum int
		port   string
	)

	if _, port, err = net.SplitHostPort(addr.String()); err != nil {
		return latest, errors.WithStack(err)
	}

	operation := ClusterOperation{
		Cluster:     c,
		AgentPort:   port,
		DialOptions: DialOptions,
	}

	counts := make(map[string]result)
	getlatest := func(c Client) (err error) {
		var (
			a *agent.Archive
		)

		if a, err = LatestDeployment(c); err != nil {
			switch err {
			case ErrNoDeployments:
				return nil
			default:
				return err
			}
		}

		key := string(a.DeploymentID)
		if r, ok := counts[key]; ok {
			counts[key] = result{a: a, count: r.count + 1}
		} else {
			counts[key] = result{a: a, count: 1}
		}

		return nil
	}

	if err = operation.Perform(operationFunc(getlatest)); err != nil {
		return latest, err
	}

	for _, v := range counts {
		if v.count > quorum {
			latest = *v.a
			quorum = v.count
		}
	}

	// check for quorum
	log.Printf("members(%d) / 2.0: min(%f) >= quorum(%f)\n", len(c.Members())-1, float64((len(c.Members())-1))/2.0, float64(quorum))
	if (float64((len(c.Members()) - 1)) / 2.0) >= float64(quorum) {
		return latest, ErrFailedDeploymentQuorum
	}

	return latest, err
}

// LatestDeployment ...
func LatestDeployment(c Client) (a *agent.Archive, err error) {
	var (
		info agent.AgentInfo
	)

	if info, err = c.Info(); err != nil {
		return nil, err
	}

	if len(info.Deployments) == 0 {
		return nil, ErrNoDeployments
	}

	return info.Deployments[0], nil
}
