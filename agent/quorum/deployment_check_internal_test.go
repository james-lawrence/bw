package quorum

import (
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

func TestFindDeployment(t *testing.T) {
	peer := &agent.Peer{Ip: "10.0.0.1"}
	current := &agent.Archive{DeploymentID: []byte("current-deployment")}
	stale := &agent.Deploy{Stage: agent.Deploy_Failed, Archive: &agent.Archive{DeploymentID: []byte("previous-deployment")}}

	t.Run("a failed earlier deploy is not the result of this one", func(t *testing.T) {
		_, err := findDeployment([]*agent.Deploy{stale}, current, peer)
		require.Error(t, err)
	})

	t.Run("no deployments means not started", func(t *testing.T) {
		_, err := findDeployment(nil, current, peer)
		require.Error(t, err)
	})

	t.Run("finds the deployment ahead of other records", func(t *testing.T) {
		d := &agent.Deploy{Stage: agent.Deploy_Deploying, Archive: current}
		found, err := findDeployment([]*agent.Deploy{stale, d}, current, peer)
		require.NoError(t, err)
		require.Equal(t, agent.Deploy_Deploying, found.Stage)
	})

	t.Run("records without an archive are skipped", func(t *testing.T) {
		_, err := findDeployment([]*agent.Deploy{{Stage: agent.Deploy_Completed}}, current, peer)
		require.Error(t, err)
	})

	t.Run("returns the failure of this deployment", func(t *testing.T) {
		d := &agent.Deploy{Stage: agent.Deploy_Failed, Archive: current}
		found, err := findDeployment([]*agent.Deploy{d, stale}, current, peer)
		require.NoError(t, err)
		require.Equal(t, agent.Deploy_Failed, found.Stage)
	})

	t.Run("returns the completion of this deployment even when newer records exist", func(t *testing.T) {
		d := &agent.Deploy{Stage: agent.Deploy_Completed, Archive: current}
		newer := &agent.Deploy{Stage: agent.Deploy_Deploying, Archive: &agent.Archive{DeploymentID: []byte("newer-deployment")}}
		found, err := findDeployment([]*agent.Deploy{newer, d}, current, peer)
		require.NoError(t, err)
		require.Equal(t, agent.Deploy_Completed, found.Stage)
	})
}
