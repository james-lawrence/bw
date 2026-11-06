package quorum

import (
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

func TestGetInfoCompleted(t *testing.T) {
	var (
		a       = agent.NewPeer("a")
		archive = &agent.Archive{DeploymentID: []byte("v1")}
		event   = agent.DeployEvent(a, &agent.Deploy{Stage: agent.Deploy_Completed, Archive: archive})
		info    = newDeployment(newSimPeers(0)).getInfo(nil, []*agent.Message{event})
	)

	require.Len(t, info.Completed, 1)
	require.Equal(t, event.Id, info.Completed[0].Id)
}
