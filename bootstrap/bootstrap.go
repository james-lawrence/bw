// Package bootstrap provides the functionality to bootstrap the agent with the
// latest deployment. the system allows for arbitrary sources to be built into
// configured by implementing a bootstrap socket which will provide an archive
// information to the agent.
package bootstrap

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type dialer interface {
	Defaults(...grpc.DialOption) dialers.Defaulted
	DialContext(ctx context.Context, options ...grpc.DialOption) (c *grpc.ClientConn, err error)
}

type cluster interface {
	GetN(n int, key []byte) []*memberlist.Node
	Local() *agent.Peer
	Peers() []*agent.Peer
	Connect() agent.ConnectResponse
}

type option func(*UntilSuccess)

// OptionMaxAttempts set maximum number of attempts.
func OptionMaxAttempts(n int) func(*UntilSuccess) {
	return func(us *UntilSuccess) {
		us.maxAttempts = n
	}
}

// OptionBackoff set strategy for backing off.
func OptionBackoff(bs backoff.Strategy) func(*UntilSuccess) {
	return func(us *UntilSuccess) {
		us.bs = bs
	}
}

// NewUntilSuccess continuously bootstraps until it succeeds or hits maximum attempts.
func NewUntilSuccess(options ...option) UntilSuccess {
	us := UntilSuccess{
		maxAttempts: math.MaxInt64, // effectively forever.
		bs: backoff.New(
			backoff.Exponential(2*time.Second),
			backoff.Maximum(time.Minute),
		),
	}

	for _, opt := range options {
		opt(&us)
	}

	return us
}

// UntilSuccess attempts to bootstrap until max attempts or success.
type UntilSuccess struct {
	maxAttempts int
	bs          backoff.Strategy
}

// Run bootstrapping process until it succeeds
func (t UntilSuccess) Run(ctx context.Context, c agent.Config, dl storage.DownloadProtocol, results chan *deployment.DeployResult) (err error) {
	var (
		sctx shell.Context
	)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	coord := deployment.New(
		c.Peer(),
		deployment.NewDirective(
			deployment.DirectiveOptionShellContext(sctx),
		),
		deployment.CoordinatorOptionRoot(c.Root),
		deployment.CoordinatorOptionKeepN(c.KeepN),
		deployment.CoordinatorOptionStorage(
			storage.New(storage.OptionProtocols(dl)),
		),
		deployment.CoordinatorOptionDispatcher(agentutil.LogDispatcher{}),
	)

	for i := 0; i < t.maxAttempts; i++ {
		if err := Bootstrap(ctx, c, coord, results); err != nil {
			log.Println(errors.Wrap(err, "bootstrap attempt failed"))
			select {
			case <-ctx.Done():
				return errors.Wrap(ctx.Err(), "bootstrap attempt failed")
			case <-time.After(t.bs.Backoff(i)):
				continue
			}
		}

		log.Println("--------------- bootstrap complete ---------------")
		return nil
	}

	log.Println("--------------- bootstrap failure ---------------")
	return errorsx.String("bootstrap failed")
}

// Bootstrap a server with the latest deploy.
func Bootstrap(ctx context.Context, c agent.Config, coord deployment.Coordinator, dresults chan *deployment.DeployResult) (err error) {
	var (
		current agent.Deploy
		latest  agent.Deploy
		deploy  *deployment.DeployResult
		results = make(chan *deployment.DeployResult)
	)

	log.Println("--------------- bootstrap attempt initiated -------------")
	defer log.Println("--------------- bootstrap attempt completed -------------")

	// Here we clone the coordinator to override some behaviours around dispatching and observations.
	coord = deployment.CloneCoordinator(
		coord,
		deployment.CoordinatorOptionDeployResults(results),
	)

	if current, err = Latest(ctx, SocketLocal(c), grpc.WithInsecure()); ignore(err) != nil {
		return errors.Wrapf(err, "latest local failed: %s", SocketLocal(c))
	}

	if latest, err = Latest(ctx, SocketQuorum(c), grpc.WithInsecure()); ignore(err) != nil {
		log.Println(errors.Wrap(err, "latest quorum failed"))
	}

	if agentutil.IsActiveDeployment(err) && agentutil.SameArchive(current.Archive, latest.Archive) {
		return errors.Wrap(err, "active deploy matches the local deployment, waiting for deployment to complete")
	}

	if err != nil && !agentutil.IsActiveDeployment(err) {
		if latest, err = getfallback(c, grpc.WithInsecure()); err != nil {
			if agentutil.IsNoDeployments(err) {
				return nil
			}

			return errors.Wrap(err, "failed to retrieve latest from fallback bootstrap services")
		}
	}

	if agentutil.SameArchive(current.Archive, latest.Archive) {
		log.Println("latest already deployed -", bw.RandomID(current.Archive.DeploymentID))
		return nil
	}

	opts := latest.Options
	deadline, cancel := context.WithTimeout(ctx, time.Duration(opts.Timeout))
	defer cancel()

	log.Println("bootstrapping", bw.RandomID(latest.Archive.DeploymentID), "with options", spew.Sdump(opts))
	if _, err = coord.Deploy(opts, latest.Archive); err != nil {
		return errors.WithStack(err)
	}

	select {
	case <-deadline.Done():
		return errors.Wrap(deadline.Err(), "failed to bootstrap timeout")
	case deploy = <-results:
	}

	if dresults != nil {
		select {
		case <-deadline.Done():
			return errors.Wrap(deadline.Err(), "failed to bootstrap timeout")
		case dresults <- deploy:
		}
	}

	if err = deploy.Error; err != nil {
		return errors.Wrap(err, "deployment failed")
	}

	// again retrieve the latest deployment information from the cluster.
	// if a deploy is ongoing or is different from the deploy we just used to bootstrap
	// we want to consider the bootstrap a failure and retry.
	if latest, err = Latest(ctx, SocketQuorum(c), grpc.WithInsecure()); err != nil && !agentutil.IsActiveDeployment(err) {
		return errors.Wrap(err, "failed to determine latest deployment from quorum, retrying")
	}

	if !agentutil.SameArchive(latest.Archive, deploy.Archive) {
		return errors.WithStack(agentutil.ErrDifferentDeployment)
	}

	return nil
}

func ignore(err error) error {
	cause := errors.Cause(err)

	switch status.Code(cause) {
	case codes.Unavailable:
		return nil
	}

	switch cause {
	case agentutil.ErrNoDeployments:
		return nil
	case agentutil.ErrActiveDeployment:
		return nil
	}

	return err
}
