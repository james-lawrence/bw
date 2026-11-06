package quorum_test

import (
	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agent/quorum"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHistory(t *testing.T) {
	t.Run("should not record heartbeats", func(t *testing.T) {
		local := agent.NewPeer("local")
		h := NewHistory()
		begin := agent.NewDeployCommand(local, &agent.DeployCommand{Command: agent.DeployCommand_Begin, Archive: &agent.Archive{}, Options: &agent.DeployOptions{}})

		require.NoError(t, h.Decode(TranscoderContext{}, begin))
		for i := 0; i < 200; i++ {
			require.NoError(t, h.Decode(TranscoderContext{}, agent.NewDeployHeartbeat(local)))
		}

		snapshot := h.Snapshot()
		require.Len(t, snapshot, 1)
		require.Equal(t, begin.Id, snapshot[0].Id)
	})

	// the buffer is emptied once when a restore starts, not for every message of it.
	t.Run("a restore replaces the history with the snapshot", func(t *testing.T) {
		local := agent.NewPeer("local")
		h := NewHistory()
		ids := func(msgs []*agent.Message) (r []string) {
			for _, m := range msgs {
				r = append(r, m.Id)
			}
			return r
		}

		before := []*agent.Message{agent.LogInfo(local, "a"), agent.LogInfo(local, "b")}
		restored := []*agent.Message{agent.LogInfo(local, "c"), agent.LogInfo(local, "d"), agent.LogInfo(local, "e"), agent.LogInfo(local, "f")}
		after := agent.LogInfo(local, "g")

		for _, m := range before {
			require.NoError(t, h.Decode(TranscoderContext{State: StateHealthy}, m))
		}
		require.NoError(t, h.Decode(TranscoderContext{State: StateRecovering}, agent.NewRestoreInitiated(local)))
		for _, m := range restored {
			require.NoError(t, h.Decode(TranscoderContext{State: StateRecovering}, m))
		}
		require.NoError(t, h.Decode(TranscoderContext{State: StateRecovering}, agent.NewRestoreCompleted(local)))
		require.Equal(t, ids(restored), ids(h.Snapshot()))

		require.NoError(t, h.Decode(TranscoderContext{State: StateHealthy}, after))
		require.Equal(t, ids(append(restored, after)), ids(h.Snapshot()))

		// a second restore starts over again.
		require.NoError(t, h.Decode(TranscoderContext{State: StateRecovering}, agent.NewRestoreInitiated(local)))
		require.NoError(t, h.Decode(TranscoderContext{State: StateRecovering}, restored[0]))
		require.Equal(t, ids(restored[:1]), ids(h.Snapshot()))

		// a restore of an empty snapshot still empties the history.
		require.NoError(t, h.Decode(TranscoderContext{State: StateRecovering}, agent.NewRestoreInitiated(local)))
		require.NoError(t, h.Decode(TranscoderContext{State: StateRecovering}, agent.NewRestoreCompleted(local)))
		require.Empty(t, h.Snapshot())
	})
}
