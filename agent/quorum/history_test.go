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
}
