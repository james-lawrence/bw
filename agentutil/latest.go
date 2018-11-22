package agentutil

import (
	"log"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/x/errorsx"
)

// DetermineLatestDeployment returns latest agent.Deploy (if any) or an error.
// If no error occurs, latest.Archive is guaranteed to be populated.
func DetermineLatestDeployment(c cluster, d dialer) (latest agent.Deploy, err error) {
	type result struct {
		deploy agent.Deploy
		count  int
	}

	var (
		votes int
	)

	counts := make(map[string]result)
	getlatest := func(c agent.Client) (err error) {
		var (
			d agent.Deploy
		)

		if d, err = AgentLatestDeployment(c); err != nil {
			switch cause := errors.Cause(err); cause {
			case ErrNoDeployments:
				return nil
			default:
				return errors.Wrap(cause, "failed while retrieving latest deployment")
			}
		}

		key := string(d.Archive.DeploymentID)
		if r, ok := counts[key]; ok {
			counts[key] = result{deploy: d, count: r.count + 1}
		} else {
			counts[key] = result{deploy: d, count: 1}
		}

		return nil
	}

	if err = NewClusterOperation(Operation(getlatest))(c, d); err != nil {
		return latest, err
	}

	for _, v := range counts {
		if v.count > votes {
			latest = v.deploy
			votes = v.count
		}
	}

	failure := errorsx.CompactMonad{}

	if len(counts) == 0 {
		failure = failure.Compact(ErrNoDeployments)
	}

	// should never happen, but if it does, guard against it.
	if latest.Archive == nil {
		failure = failure.Compact(errors.Wrap(ErrNoDeployments, "archive missing in deploy"))
	}

	if !quorum(c, votes) {
		failure = failure.Compact(ErrFailedDeploymentQuorum)
	}

	// TODO check archive for a deployment if failure is not nil.
	return latest, failure.Cause()
}

func quorum(c cluster, votes int) bool {
	// subtract one as to not count the current node as part of the quorum.
	peers := len(c.Peers()) - 1
	minRatio := float64(peers) / 2.0
	log.Printf("votes(%f) >= members(%d) / 2.0: min(%f)\n", float64(votes), peers, minRatio)
	return float64(votes) >= minRatio
}

// QuorumLatestDeployment determines the latest deployment by asking the agents
// who are in the raft cluster what the latest deployment is.
func QuorumLatestDeployment(c cluster, d dialer) (z agent.Deploy, err error) {
	var (
		i      agent.InfoResponse
		client agent.Client
	)

	if client, err = agent.NewQuorumDialer(d).Dial(c); err != nil {
		return z, err
	}

	if i, err = client.QuorumInfo(); err != nil {
		return z, errorsx.Compact(err, client.Close())
	}

	if err = client.Close(); err != nil {
		return z, err
	}

	if i.Deployed == nil {
		return z, errors.WithStack(ErrNoDeployments)
	}

	if i.Mode == agent.InfoResponse_Deploying {
		return agent.Deploy{
			Stage:   agent.Deploy_Deploying,
			Archive: i.Deploying.Archive,
			Options: i.Deploying.Options,
		}, errors.WithStack(ErrActiveDeployment)
	}

	return agent.Deploy{
		Stage:   agent.Deploy_Completed,
		Archive: i.Deployed.Archive,
		Options: i.Deployed.Options,
	}, err
}

// LocalLatestDeployment determines the latest successful local deployment.
func LocalLatestDeployment(local agent.Peer, d dialer) (a agent.Deploy, err error) {
	var (
		c agent.Client
	)

	if c, err = d.Dial(local); err != nil {
		return a, errors.Wrap(err, "failed to connect to local server")
	}

	a, err = AgentLatestDeployment(c)

	return a, errorsx.Compact(err, c.Close())
}

// AgentLatestDeployment determines the latest deployment of a given agent.
func AgentLatestDeployment(c agent.Client) (a agent.Deploy, err error) {
	var (
		info agent.StatusResponse
	)

	if info, err = c.Info(); err != nil {
		return a, errors.Wrap(err, "failed to retrieve latest deployment")
	}

	if len(info.Deployments) == 0 {
		return a, errors.WithStack(ErrNoDeployments)
	}

	for _, d := range info.Deployments {
		if d.Stage == agent.Deploy_Completed {
			return *d, nil
		}
	}

	// no successful deploys
	return a, errors.WithStack(ErrNoDeployments)
}
