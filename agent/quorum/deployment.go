package quorum

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/logx"

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

func (t *deployment) Decode(_ TranscoderContext, m agent.Message) error {
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
		if !atomic.CompareAndSwapInt32(&t.deploying, none, deploying) {
			return errors.New(fmt.Sprint("deploy already in progress"))
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
func (t *deployment) deploy(d agent.Dialer, dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) (err error) {
	return t.d.Deploy(d, dopts, a, peers...)
}

// Cancel a ongoing deploy.
func (t *deployment) cancel(ctx context.Context, d agent.Dialer, sm stateMachine) (err error) {
	if err = agentutil.Cancel(t.c, d); err != nil {
		return err
	}

	return sm.Dispatch(ctx, agentutil.DeployCommand(t.c.Local(), agentutil.DeployCommandCancel("")))
}

func (t *deployment) determineLatestDeploy(ctx context.Context, d agent.Dialer, sm stateMachine) (err error) {
	var (
		deploy agent.Deploy
	)

	last := t.getLastSuccessfulDeploy()
	if last != nil {
		return nil
	}

	log.Println("leadership change detected missing successful deploy, attempting to recover")
	if deploy, err = agentutil.DetermineLatestDeployment(t.c, d); err != nil {
		return err
	}

	return sm.Dispatch(ctx,
		agentutil.DeployCommand(t.c.Local(), agent.DeployCommand{
			Command: agent.DeployCommand_Done,
			Archive: deploy.Archive,
			Options: deploy.Options,
		}),
	)
}

func (t *deployment) restartActiveDeploy(ctx context.Context, d agent.Dialer, sm stateMachine) (err error) {
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
			return errors.Wrap(err, "log restart detection failure")
		}

		if err = t.cancel(ctx, d, sm); err != nil {
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

		if err = t.deploy(d, *dc.Options, *dc.Archive); err != nil {
			return errors.Wrap(err, "deploy failure")
		}
	}

	return nil
}