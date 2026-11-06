package quorum

import (
	"context"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	deployments "github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// Initializer for the state machine.
type Initializer interface {
	Initialize(agent.Dispatcher) error
}

// NewMachine the work started by the machine, such as deploys, is bound to the context.
func NewMachine(ctx context.Context, l *agent.Peer, rp *raft.Raft, inits ...Initializer) *StateMachine {
	return &StateMachine{
		ctx:   ctx,
		l:     l,
		state: rp,
		inits: inits,
	}
}

// StateMachine wraps the raft protocol giving more convient access to the protocol.
type StateMachine struct {
	ctx   context.Context // bounds the work started by the machine, such as deploys. cancelled when leadership ends.
	l     *agent.Peer
	state *raft.Raft
	inits []Initializer
}

func (t *StateMachine) initialize() (err error) {
	for _, init := range t.inits {
		if err = init.Initialize(t); err != nil {
			return err
		}
	}

	return nil
}

// Leader returns the current leader.
func (t *StateMachine) Leader() *agent.Peer {
	return t.l
}

// State returns the state of the raft cluster.
func (t *StateMachine) State() raft.RaftState {
	return t.state.State()
}

func (t *StateMachine) Deploy(ctx context.Context, c cluster, dialer dialers.Defaults, by string, dopts *agent.DeployOptions, archive *agent.Archive, peers ...*agent.Peer) (err error) {
	var (
		filter deployments.Filter
	)

	qd := dialers.NewQuorum(c, dialer.Defaults()...)
	d := agentutil.NewDispatcher(qd)

	// the coordinator stops when leadership ends, or an event shows another coordinator has taken over.
	coordinator, cancel := context.WithCancelCause(t.ctx)

	cmd := agent.DeployCommandBegin(by, archive, dopts)

	if err = d.Dispatch(ctx, agent.NewDeployCommand(c.Local(), cmd)); err != nil {
		cancel(err)
		return err
	}

	filter = deployments.AlwaysMatch
	if len(peers) > 0 {
		filter = deployments.Peers(peers...)
	}

	options := []deployments.Option{
		deployments.DeployOptionChecker(deployments.OperationFunc(check(dialer))),
		deployments.DeployOptionDeployer(deployments.OperationFunc(deploy(by, dopts, archive, dialer))),
		deployments.DeployOptionFilter(filter),
		deployments.DeployOptionPartitioner(bw.ConstantPartitioner(dopts.Concurrency)),
		deployments.DeployOptionIgnoreFailures(dopts.IgnoreFailures),
		deployments.DeployOptionAttempts(dopts.Attempts),
		deployments.DeployOptionTimeout(time.Duration(dopts.Timeout)),
		deployments.DeployOptionHeartbeatFrequency(time.Duration(dopts.Heartbeat)),
		deployments.DeployOptionMonitor(deployments.NewMonitor(
			deployments.MonitorTicklerEvent(c.Local(), qd, archive.GetDeploymentID(), cancel),
			deployments.MonitorTicklerPeriodicAuto(time.Minute),
		)),
	}

	// At this point the deploy could take awhile, so we shunt it into the background.
	go func() {
		defer cancel(nil)

		dcmd := agent.DeployCommandFailed(by, archive.DeployOption, dopts.DeployOption)
		_, success := deployments.RunDeploy(coordinator, c.Local(), c, d, options...)

		// a stopped coordinator has been superseded (restarted, cancelled or leadership moved), whatever
		// replaced it records the outcome. reporting would end the replacement: the deployment is the same.
		if cause := context.Cause(coordinator); cause != nil {
			log.Println("deploy coordinator stopped, not reporting a result:", cause)
			return
		}

		if success {
			dcmd = agent.DeployCommandDone(by, archive.DeployOption, dopts.DeployOption)
		}

		if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
			log.Println("deployment complete", spew.Sdump(&dcmd))
		}
		errorsx.Log(dispatchDeployResult(context.Background(), d, c.Local(), dcmd))
	}()

	return nil
}

// dispatchDeployResult records the outcome of a deployment. clients and the cluster
// wait on this message to consider the deploy finished, so retry until it's delivered.
func dispatchDeployResult(ctx context.Context, d agent.Dispatcher, local *agent.Peer, dcmd *agent.DeployCommand) error {
	ctx, done := context.WithTimeout(ctx, time.Minute)
	defer done()

	return agentutil.ReliableDispatch(ctx, d, agent.NewDeployCommand(local, dcmd))
}

func check(d dialers.Defaults) func(ctx context.Context, n *agent.Peer) (*agent.Deploy, error) {
	return func(ctx context.Context, n *agent.Peer) (_d *agent.Deploy, err error) {
		var (
			c    *grpc.ClientConn
			info *agent.StatusResponse
		)

		if c, err = dialers.NewDirect(agent.RPCAddress(n)).DialContext(ctx, d.Defaults()...); err != nil {
			return _d, err
		}

		defer c.Close()

		if info, err = agent.NewConn(c).Info(ctx); err != nil {
			return _d, err
		}

		if len(info.Deployments) > 0 {
			return info.Deployments[0], nil
		}

		return &agent.Deploy{
			Stage: agent.Deploy_Completed,
		}, nil
	}
}

func deploy(by string, dopts *agent.DeployOptions, archive *agent.Archive, d dialers.Defaults) func(ctx context.Context, n *agent.Peer) (*agent.Deploy, error) {
	return func(ctx context.Context, n *agent.Peer) (_d *agent.Deploy, err error) {
		var (
			c *grpc.ClientConn
		)

		if c, err = dialers.NewDirect(agent.RPCAddress(n)).DialContext(ctx, d.Defaults()...); err != nil {
			return _d, err
		}
		defer c.Close()

		return agent.NewConn(c).Deploy(ctx, by, dopts, archive)
	}
}

// Dispatch a message to the WAL.
func (t *StateMachine) Dispatch(ctx context.Context, messages ...*agent.Message) (err error) {
	for _, m := range messages {
		if err = t.writeWAL(m, 10*time.Second); err != nil {
			return err
		}
	}

	return nil
}

func (t *StateMachine) writeWAL(m *agent.Message, d time.Duration) (err error) {
	var (
		encoded []byte
		future  raft.ApplyFuture
		ok      bool
	)

	if encoded, err = proto.Marshal(m); err != nil {
		return errors.WithStack(err)
	}

	// write the event to the WAL.
	future = t.state.Apply(encoded, 10*time.Second)

	if err = future.Error(); err != nil {
		return errors.WithStack(err)
	}

	if err, ok = future.Response().(error); ok {
		return errors.WithStack(err)
	}

	return err
}
