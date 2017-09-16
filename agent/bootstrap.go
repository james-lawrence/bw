package agent

import (
	"net"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
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

// DetermineLatestArchive ...
func DetermineLatestArchive(addr net.Addr, c clustering.Cluster, DialOptions ...grpc.DialOption) (latest agent.Archive, err error) {
	type result struct {
		a     *agent.Archive
		count int
	}

	var (
		max  int
		port string
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
		if v.count > max {
			latest = *v.a
			max = v.count
		}
	}

	// check for quorum
	if (len(c.Members()) / 2.0) >= max {
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
