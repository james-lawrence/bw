package quorum

import (
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"
	"github.com/stretchr/testify/require"
)

func TestScanLatest(t *testing.T) {
	var (
		a  = agent.NewPeer("a")
		b  = agent.NewPeer("b")
		v1 = &agent.Archive{DeploymentID: []byte("v1")}
		v2 = &agent.Archive{DeploymentID: []byte("v2")}
	)

	newJournal := func(t *testing.T, ms ...*agent.Message) *journal.Journal {
		t.Helper()

		j, err := journal.New(t.TempDir())
		require.NoError(t, err)
		t.Cleanup(func() { j.Close() })
		require.NoError(t, j.Dispatch(t.Context(), ms...))

		return j
	}

	scan := func(t *testing.T, ms ...*agent.Message) latest {
		t.Helper()

		state, err := scanLatest(t.Context(), newJournal(t, ms...))
		require.NoError(t, err)
		return state
	}

	begin := func(p *agent.Peer, a *agent.Archive) *agent.Message {
		return agent.NewDeployCommand(p, agent.DeployCommandBegin("tester", a, &agent.DeployOptions{}))
	}

	completed := func(p *agent.Peer, a *agent.Archive) *agent.Message {
		return agent.DeployEvent(p, &agent.Deploy{Stage: agent.Deploy_Completed, Archive: a})
	}

	deployed := func(state latest, node string) []byte {
		return state.deployed[node].GetDeploy().GetArchive().GetDeploymentID()
	}

	t.Run("history", func(t *testing.T) {
		t.Run("is empty for an empty journal", func(t *testing.T) {
			state := scan(t)
			require.Empty(t, state.history)
			require.Empty(t, state.deployed)
		})

		t.Run("holds the messages in order", func(t *testing.T) {
			ms := []*agent.Message{agent.LogInfo(a, "1"), agent.LogInfo(a, "2"), agent.LogInfo(a, "3")}
			require.Equal(t, ids(ms), ids(scan(t, ms...).history))
		})

		t.Run("does not record heartbeats", func(t *testing.T) {
			first := begin(a, v1)
			ms := []*agent.Message{first}
			for i := 0; i < 200; i++ {
				ms = append(ms, agent.NewDeployHeartbeat(a))
			}

			require.Equal(t, []string{first.Id}, ids(scan(t, ms...).history))
		})

		t.Run("is the latest deploy", func(t *testing.T) {
			second := begin(a, v2)
			tail := agent.LogInfo(a, "tail")
			state := scan(t, begin(a, v1), agent.LogInfo(a, "old"), second, tail)

			require.Equal(t, []string{second.Id, tail.Id}, ids(state.history))
		})

		t.Run("is limited to the newest messages", func(t *testing.T) {
			ms := []*agent.Message{begin(a, v1)}
			for i := 0; i < 300; i++ {
				ms = append(ms, agent.LogInfo(a, "noise"))
			}

			history := scan(t, ms...).history
			require.Len(t, history, historyLimit)
			require.Equal(t, ids(ms[len(ms)-historyLimit:]), ids(history))
		})

		t.Run("holds a message once", func(t *testing.T) {
			m := agent.LogInfo(a, "twice")
			require.Equal(t, []string{m.Id}, ids(scan(t, m, m).history))
		})

		t.Run("does not hold hidden messages", func(t *testing.T) {
			hidden := agent.LogInfo(a, "hidden")
			hidden.Hidden = true
			shown := agent.LogInfo(a, "shown")

			require.Equal(t, []string{shown.Id}, ids(scan(t, hidden, shown).history))
		})
	})

	t.Run("deployed", func(t *testing.T) {
		t.Run("records the deployment each node completed", func(t *testing.T) {
			state := scan(t, completed(a, v1), completed(b, v2))
			require.Len(t, state.deployed, 2)
			require.Equal(t, v1.DeploymentID, deployed(state, "a"))
			require.Equal(t, v2.DeploymentID, deployed(state, "b"))
		})

		t.Run("a newer deployment replaces the previous one", func(t *testing.T) {
			require.Equal(t, v2.DeploymentID, deployed(scan(t, completed(a, v1), completed(a, v2)), "a"))
		})

		t.Run("deploys that arent complete leave what the node had", func(t *testing.T) {
			state := scan(t,
				completed(a, v1),
				agent.DeployEvent(a, &agent.Deploy{Stage: agent.Deploy_Deploying, Archive: v2}),
				agent.DeployEvent(a, &agent.Deploy{Stage: agent.Deploy_Failed, Archive: v2}),
				// a node failing its first deploy has nothing.
				agent.DeployEvent(b, &agent.Deploy{Stage: agent.Deploy_Failed, Archive: v2}),
			)

			require.Len(t, state.deployed, 1)
			require.Equal(t, v1.DeploymentID, deployed(state, "a"))
		})

		t.Run("a completed deploy without a deployment is ignored", func(t *testing.T) {
			require.Empty(t, scan(t, agent.DeployEvent(a, &agent.Deploy{Stage: agent.Deploy_Completed})).deployed)
		})

		t.Run("nodes are kept when a deploy begins", func(t *testing.T) {
			state := scan(t, completed(a, v1), begin(b, v2))
			require.Equal(t, v1.DeploymentID, deployed(state, "a"))
		})

		t.Run("keeps the event so the peer and the time are known", func(t *testing.T) {
			event := completed(a, v1)
			state := scan(t, event)

			require.Equal(t, event.Id, state.deployed["a"].Id)
			require.Equal(t, a.Name, state.deployed["a"].GetPeer().GetName())
			require.Equal(t, agent.MessageTime(event).UnixMilli(), agent.MessageTime(state.deployed["a"]).UnixMilli())
		})

		t.Run("the events are ordered by node name", func(t *testing.T) {
			c := agent.NewPeer("c")
			require.Equal(t, []string{"a", "b", "c"}, func() (names []string) {
				for _, m := range scan(t, completed(c, v1), completed(a, v1), completed(b, v1)).completed() {
					names = append(names, m.GetPeer().GetName())
				}
				return names
			}())
		})

		t.Run("nodes are kept until they depart", func(t *testing.T) {
			state := scan(t, completed(a, v1), completed(b, v1), agent.NodeEvent(a, agent.Message_Joined))
			require.Len(t, state.deployed, 2)

			state = scan(t, completed(a, v1), completed(b, v1), agent.NodeEvent(a, agent.Message_Departed))
			require.Len(t, state.deployed, 1)
			require.Contains(t, state.deployed, "b")
		})
	})
}
