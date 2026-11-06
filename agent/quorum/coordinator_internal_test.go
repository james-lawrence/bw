package quorum

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/journal"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/stretchr/testify/require"
)

// capture is a state machine recording what is dispatched to it.
type capture struct {
	m        sync.Mutex
	messages []*agent.Message
}

func (t *capture) State() raft.RaftState { return raft.Leader }
func (t *capture) Leader() *agent.Peer   { return nil }
func (t *capture) Deploy(context.Context, cluster, dialers.Defaults, string, *agent.DeployOptions, *agent.Archive, ...*agent.Peer) error {
	return nil
}

func (t *capture) Dispatch(_ context.Context, m ...*agent.Message) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.messages = append(t.messages, m...)
	return nil
}

func TestRestartActiveDeploy(t *testing.T) {
	t.Run("the restart names the coordinator and carries the deployment", func(t *testing.T) {
		var (
			previous = agent.NewPeer("previous-leader")
			leader   = agent.NewPeer("leader")
			c        = simpeers{local: leader}
			d        = newDeployment(c)
			sm       = &capture{}
			archive  = &agent.Archive{DeploymentID: []byte("d1")}
			options  = &agent.DeployOptions{Concurrency: 3}
		)

		// the previous leader began the deploy.
		begin := agent.NewDeployCommand(previous, agent.DeployCommandBegin("tester", archive, options))
		require.NoError(t, d.Decode(TranscoderContext{State: StateHealthy}, begin))
		require.Equal(t, previous.Name, d.getCoordinator().Name)

		ctx, done := context.WithTimeout(t.Context(), 10*time.Second)
		defer done()

		// the redeploy fails, there is no network. only the events matter.
		_ = d.restartActiveDeploy(ctx, dialers.NewDefaults(), sm)

		var restart *agent.DeployCommand
		for _, m := range sm.messages {
			if dc := m.GetDeployCommand(); dc.GetCommand() == agent.DeployCommand_Restart {
				require.Equal(t, leader.Name, m.GetPeer().Name, "the message comes from the new leader")
				restart = dc
			}
		}

		require.NotNil(t, restart)
		require.Equal(t, previous.Name, restart.GetPrevious().GetName())
		require.Equal(t, "tester", restart.GetInitiator())
		require.Equal(t, archive.DeploymentID, restart.GetArchive().GetDeploymentID())
		require.EqualValues(t, 3, restart.GetOptions().GetConcurrency())
	})

	t.Run("the coordinator is forgotten when the deploy is done", func(t *testing.T) {
		var (
			leader  = agent.NewPeer("leader")
			d       = newDeployment(simpeers{local: leader})
			archive = &agent.Archive{DeploymentID: []byte("d1")}
		)

		require.NoError(t, d.Decode(TranscoderContext{State: StateHealthy}, agent.NewDeployCommand(leader, agent.DeployCommandBegin("tester", archive, &agent.DeployOptions{}))))
		require.NotNil(t, d.getCoordinator())

		require.NoError(t, d.Decode(TranscoderContext{State: StateHealthy}, agent.NewDeployCommand(leader, agent.DeployCommandDone("tester", archive.DeployOption))))
		require.Nil(t, d.getCoordinator())
	})
}

func TestLeadingBindsTheMachine(t *testing.T) {
	newQuorum := func(t *testing.T) *Quorum {
		jrnl, err := journal.New(t.TempDir())
		require.NoError(t, err)
		t.Cleanup(func() { jrnl.Close() })

		q := New(jrnl, newSimPeers(0), &recorder{}, nil, raftutil.Protocol{})
		return &q
	}

	t.Run("stepping down cancels the work of the machine", func(t *testing.T) {
		q := newQuorum(t)
		q.leading(&raft.Raft{})

		sm := q.proxy().(*StateMachine)
		require.NoError(t, sm.ctx.Err())

		q.stepdown(errors.New("leadership lost"))
		require.Error(t, sm.ctx.Err())
		require.EqualError(t, context.Cause(sm.ctx), "leadership lost")

		// nothing to do when not leading.
		q.stepdown(errors.New("again"))
		require.EqualError(t, context.Cause(sm.ctx), "leadership lost")
	})

	t.Run("a new term cancels the work of the previous one", func(t *testing.T) {
		q := newQuorum(t)
		q.leading(&raft.Raft{})
		first := q.proxy().(*StateMachine)

		q.leading(&raft.Raft{})
		second := q.proxy().(*StateMachine)

		require.NotSame(t, first, second)
		require.Error(t, first.ctx.Err())
		require.NoError(t, second.ctx.Err())
	})

	t.Run("observing leadership again keeps the work running", func(t *testing.T) {
		q := newQuorum(t)
		r := &raft.Raft{}
		q.leading(r)
		sm := q.proxy().(*StateMachine)

		q.leading(r)
		require.Same(t, sm, q.proxy())
		require.NoError(t, sm.ctx.Err())
	})
}
