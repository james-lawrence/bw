package agentutil

import (
	"log"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/agent"

	"google.golang.org/grpc"
)

// DetermineLatestArchive ...
func DetermineLatestArchive(c cluster, doptions ...grpc.DialOption) (latest agent.Archive, err error) {
	type result struct {
		a     *agent.Archive
		count int
	}

	var (
		quorum int
	)

	counts := make(map[string]result)
	getlatest := func(c agent.Client) (err error) {
		var (
			a *agent.Archive
		)

		if a, err = LatestDeployment(c); err != nil {
			switch cause := errors.Cause(err); cause {
			case ErrNoDeployments:
				return nil
			default:
				return errors.Wrap(cause, "failed while retrieving latest deployment")
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

	if err = NewClusterOperation(Operation(getlatest))(c, doptions...); err != nil {
		return latest, err
	}

	if len(counts) == 0 {
		return latest, ErrNoDeployments
	}

	for _, v := range counts {
		if v.count > quorum {
			latest = *v.a
			quorum = v.count
		}
	}

	peers := c.Peers()
	// check for quorum
	log.Printf("quorum(%f) < members(%d) / 2.0: min(%f)\n", float64(quorum), len(peers)-1, float64((len(peers)-1))/2.0)
	if float64(quorum) < float64((len(peers)-1))/2.0 {
		return latest, ErrFailedDeploymentQuorum
	}

	return latest, err
}

// LatestDeployment ...
func LatestDeployment(c agent.Client) (a *agent.Archive, err error) {
	var (
		info agent.Status
	)

	if info, err = c.Info(); err != nil {
		return nil, errors.Wrap(err, "latest deployment failed")
	}

	if len(info.Deployments) == 0 {
		return nil, errors.WithStack(ErrNoDeployments)
	}

	return info.Deployments[0], nil
}
