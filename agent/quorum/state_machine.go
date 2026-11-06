package quorum

import (
	"bytes"
	"context"
	"log"
	"slices"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	// maxInflightWrites bounds the writes waiting on raft at once. beyond that callers are told to back off
	// rather than piling up behind a leader that can't keep up.
	maxInflightWrites = 64
	// writeAdmitTimeout how long a write waits for a slot, and for raft to accept it, before the caller is
	// told to back off. kept well below the callers' own deadlines so they hear about it.
	writeAdmitTimeout = 2 * time.Second
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

		inflight: make(chan struct{}, maxInflightWrites),
	}
}

// StateMachine wraps the raft protocol giving more convient access to the protocol.
type StateMachine struct {
	ctx   context.Context // bounds the work started by the machine, such as deploys. cancelled when leadership ends.
	l     *agent.Peer
	state *raft.Raft
	inits []Initializer

	inflight chan struct{} // admission for writes to the WAL.
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
		deployments.DeployOptionChecker(deployments.OperationFunc(check(dialer, archive))),
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

func check(d dialers.Defaults, archive *agent.Archive) func(ctx context.Context, n *agent.Peer) (*agent.Deploy, error) {
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

		return findDeployment(info.Deployments, archive, n)
	}
}

// findDeployment the node's record of the given deployment. records of other deployments (e.g. an earlier failed
// deploy) say nothing about this one, until the node has recorded it the deployment hasn't started.
func findDeployment(deployments []*agent.Deploy, archive *agent.Archive, n *agent.Peer) (*agent.Deploy, error) {
	i := slices.IndexFunc(deployments, func(d *agent.Deploy) bool {
		return d.GetArchive() != nil && bytes.Equal(d.Archive.DeploymentID, archive.DeploymentID)
	})

	if i < 0 {
		return nil, errors.Errorf("deployment %s not found on %s", bw.RandomID(archive.DeploymentID), n.Ip)
	}

	return deployments[i], nil
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
		if err = t.writeWAL(ctx, m, writeAdmitTimeout); err != nil {
			return err
		}
	}

	return nil
}

// writeWAL writes the message to the WAL. when raft can't take the write promptly the caller is told to
// back off (codes.ResourceExhausted) instead of being held until its own deadline.
func (t *StateMachine) writeWAL(ctx context.Context, m *agent.Message, d time.Duration) (err error) {
	var (
		encoded []byte
		future  raft.ApplyFuture
		ok      bool
	)

	if encoded, err = proto.Marshal(m); err != nil {
		return errors.WithStack(err)
	}

	// bound the writes in flight.
	admit := time.NewTimer(d)
	defer admit.Stop()

	select {
	case t.inflight <- struct{}{}:
		defer func() { <-t.inflight }()
	case <-admit.C:
		return status.Error(codes.ResourceExhausted, "too many writes in flight, back off")
	case <-ctx.Done():
		return errors.WithStack(ctx.Err())
	}

	// write the event to the WAL.
	future = t.state.Apply(encoded, d)

	if err = future.Error(); err != nil {
		return backpressure(err)
	}

	if err, ok = future.Response().(error); ok {
		return errors.WithStack(err)
	}

	return err
}

// backpressure translates raft errors into statuses that tell the caller what to do.
func backpressure(err error) error {
	switch {
	case errors.Is(err, raft.ErrEnqueueTimeout):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, raft.ErrNotLeader), errors.Is(err, raft.ErrLeadershipLost), errors.Is(err, raft.ErrRaftShutdown):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return errors.WithStack(err)
	}
}
