package proxy

import (
	"context"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/logx"
)

type clusterx interface {
	GetN(n int, key []byte) []*memberlist.Node
	Local() *agent.Peer
	Peers() []*agent.Peer
}

// NewProxy ...
func NewProxy(c clusterx) Proxy {
	return Proxy{
		c: c,
	}
}

// Proxy - implements the deployer.
type Proxy struct {
	c clusterx
}

// Deploy the given archive to the specified peers.
// The deploy itself is run asychronously as they take awhile, letting callers
// continue on. But it will error out if there are issues initiating the deploy.
// such as another deploy is currently running.
func (t Proxy) Deploy(dialer dialers.Defaults, dopts *agent.DeployOptions, archive *agent.Archive, peers ...*agent.Peer) (err error) {
	var (
		filter deployment.Filter
	)

	qd := dialers.NewQuorum(t.c, dialer.Defaults()...)
	d := agentutil.NewDispatcher(qd)

	cmd := &agent.DeployCommand{
		Command: agent.DeployCommand_Begin,
		Options: dopts,
		Archive: archive,
	}

	if err = d.Dispatch(context.Background(), agentutil.DeployCommand(t.c.Local(), cmd)); err != nil {
		return err
	}

	filter = deployment.AlwaysMatch
	if len(peers) > 0 {
		filter = deployment.Peers(peers...)
	}

	options := []deployment.Option{
		deployment.DeployOptionChecker(deployment.OperationFunc(check(dialer))),
		deployment.DeployOptionDeployer(deployment.OperationFunc(deploy(dopts, archive, dialer))),
		deployment.DeployOptionFilter(filter),
		deployment.DeployOptionPartitioner(bw.ConstantPartitioner(dopts.Concurrency)),
		deployment.DeployOptionIgnoreFailures(dopts.IgnoreFailures),
		deployment.DeployOptionTimeoutGrace(time.Duration(dopts.Timeout)),
		deployment.DeployOptionMonitor(deployment.NewMonitor(
			deployment.MonitorTicklerEvent(t.c.Local(), qd),
			deployment.MonitorTicklerPeriodicAuto(time.Minute),
		)),
	}

	// At this point the deploy could take awhile, so we shunt it into the background.
	go func() {
		dcmd := agent.DeployCommand{Command: agent.DeployCommand_Failed, Archive: archive, Options: dopts}
		if _, success := deployment.RunDeploy(t.c.Local(), t.c, d, options...); success {
			dcmd = agent.DeployCommand{Command: agent.DeployCommand_Done, Archive: archive, Options: dopts}
		}

		if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
			log.Println("deployment complete", spew.Sdump(&dcmd))
		}
		logx.MaybeLog(d.Dispatch(context.Background(), agentutil.DeployCommand(t.c.Local(), &dcmd)))
	}()

	return nil
}
