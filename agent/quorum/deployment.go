package quorum

import (
	"context"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/logx"

	"github.com/pkg/errors"
)

const (
	none int32 = iota
	deploying
)

func newDeployment(d deployer, c cluster) *deployment {
	return &deployment{
		m: &sync.RWMutex{},
		d: d,
		c: c,
	}
}

type deployment struct {
	c                    cluster
	d                    deployer
	deploying            int32                // is a deploy process in progress.
	runningDeploy        *agent.DeployCommand // currently active deployment.
	lastSuccessfulDeploy *agent.DeployCommand // used for bootstrapping and recovering when a deploy proxy fails.
	m                    *sync.RWMutex
}

func (t *deployment) Encode(dst io.Writer) error {
	return nil
}

func (t *deployment) Decode(ctx TranscoderContext, m *agent.Message) error {
	var (
		dc *agent.DeployCommand
	)

	switch event := m.GetEvent().(type) {
	case *agent.Message_DeployCommand:
		dc = event.DeployCommand
	default:
		return nil
	}

	debugx.Println("deploy command received", dc.Command.String())
	defer debugx.Println("deploy command processed", dc.Command.String())

	switch dc.Command {
	case agent.DeployCommand_Begin:
		if ctx.State != StateRecovering && !atomic.CompareAndSwapInt32(&t.deploying, none, deploying) {
			return errors.New("deploy already in progress")
		}

		t.m.Lock()
		t.runningDeploy = dc
		t.m.Unlock()
	case agent.DeployCommand_Done:
		atomic.SwapInt32(&t.deploying, none)
		t.m.Lock()
		t.lastSuccessfulDeploy = dc
		t.runningDeploy = nil
		t.m.Unlock()
	default:
		atomic.SwapInt32(&t.deploying, none)
	}

	return nil
}

func (t *deployment) getInfo(leader *agent.Peer) agent.InfoResponse {
	t.m.RLock()
	defer t.m.RUnlock()

	m := agent.InfoResponse_None
	if atomic.LoadInt32(&t.deploying) == deploying {
		m = agent.InfoResponse_Deploying
	}

	return agent.InfoResponse{
		Mode:      m,
		Deploying: t.runningDeploy,
		Deployed:  t.lastSuccessfulDeploy,
		Leader:    leader,
		Quorum:    agent.QuorumPeers(t.c),
	}
}

func (t *deployment) getLastSuccessfulDeploy() *agent.DeployCommand {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.lastSuccessfulDeploy
}

func (t *deployment) getRunningDeploy() *agent.DeployCommand {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.runningDeploy
}

// Deploy trigger a deploy.
func (t *deployment) deploy(d dialers.Defaults, dopts *agent.DeployOptions, a *agent.Archive, peers ...*agent.Peer) (err error) {
	return t.d.Deploy(d, dopts, a, peers...)
}

// Cancel a ongoing deploy.
func (t *deployment) cancel(ctx context.Context, req *agent.CancelRequest, d dialers.Defaults, sm stateMachine) (err error) {
	if err = agentutil.Cancel(t.c, d.Defaults()); err != nil {
		return err
	}

	return sm.Dispatch(ctx, agentutil.DeployCommand(t.c.Local(), agentutil.DeployCommandCancel(req.Initiator)))
}

func (t *deployment) determineLatestDeploy(ctx context.Context, d dialers.Defaults, sm stateMachine) (err error) {
	var (
		deploy *agent.Deploy
	)

	last := t.getLastSuccessfulDeploy()
	if last != nil {
		return nil
	}

	log.Println("leadership change detected missing successful deploy, attempting to recover")
	if deploy, err = agentutil.DetermineLatestDeployment(t.c, d.Defaults()); err != nil {
		return err
	}

	return sm.Dispatch(ctx,
		agentutil.DeployCommand(t.c.Local(), &agent.DeployCommand{
			Command: agent.DeployCommand_Done,
			Archive: deploy.Archive,
			Options: deploy.Options,
		}),
	)
}

func (t *deployment) restartActiveDeploy(ctx context.Context, d dialers.Defaults, sm stateMachine) (err error) {
	var (
		dc *agent.DeployCommand
	)

	if dc = t.getRunningDeploy(); dc != nil && dc.Options != nil && dc.Archive != nil {
		err = sm.Dispatch(
			ctx,
			agentutil.LogEvent(t.c.Local(), "detected new leader during an active deployment, attempting to recover"),
			agentutil.LogEvent(t.c.Local(), "attempting to cancel running deployments"),
		)

		if err != nil {
			return errors.Wrap(err, "unable to write restart events due to leadership change")
		}

		if err = t.cancel(ctx, &agent.CancelRequest{}, d, sm); err != nil {
			msg := agentutil.LogEvent(t.c.Local(), "failed to cancel running deployments")
			logx.MaybeLog(sm.Dispatch(ctx, msg))
			return errors.Wrap(err, "cancellation failure")
		}

		err = sm.Dispatch(
			ctx,
			agentutil.LogEvent(t.c.Local(), "restarting deploy"),
			agentutil.DeployCommand(t.c.Local(), agentutil.DeployCommandRestart()),
		)

		if err != nil {
			return errors.Wrap(err, "restart command failure")
		}

		if err = t.deploy(d, dc.Options, dc.Archive); err != nil {
			return errors.Wrap(err, "deploy failure")
		}
	}

	return nil
}
