package ux

import (
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/logrusorgru/aurora"
	"github.com/stretchr/testify/require"
)

func TestColorize(t *testing.T) {
	t.Run("properly format messages", func(t *testing.T) {
		au := aurora.NewAurora(true)
		m := agent.PeerEvent(agent.NewPeerZero(), agent.StampZero)
		evt := agent.LogMessage(m, agent.Log_INFO, "DERP DERP")
		require.Equal(t, "Jan  1 00:00:00 00000000-0000-0000-0000-000000000000:0.0.0.0      - INFO - DERP DERP", typography(au, "%[1]s - %[2]s - %[3]s", m, evt).String())
	})
}
