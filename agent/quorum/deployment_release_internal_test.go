package quorum

import (
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

func TestDeploymentReleasesOnlyNamedDeployment(t *testing.T) {
	var (
		leader = agent.NewPeer("leader")
		ours   = &agent.Archive{DeploymentID: []byte("ours")}
		other  = &agent.Archive{DeploymentID: []byte("other")}
		opts   = &agent.DeployOptions{}
		ctx    = TranscoderContext{State: StateHealthy}
	)

	begin := func(a *agent.Archive) *agent.Message {
		return agent.NewDeployCommand(leader, agent.DeployCommandBegin("tester", a, opts))
	}

	tests := map[string]struct {
		cmd      *agent.DeployCommand
		released bool
	}{
		"done for another deployment":    {cmd: agent.DeployCommandDone("tester", other.DeployOption, opts.DeployOption)},
		"failed for another deployment":  {cmd: agent.DeployCommandFailed("tester", other.DeployOption, opts.DeployOption)},
		"restart for another deployment": {cmd: agent.DeployCommandRestart(leader, &agent.DeployCommand{Archive: other, Options: opts})},
		"done for the deployment":        {cmd: agent.DeployCommandDone("tester", ours.DeployOption, opts.DeployOption), released: true},
		"failed for the deployment":      {cmd: agent.DeployCommandFailed("tester", ours.DeployOption, opts.DeployOption), released: true},
		"restart for the deployment":     {cmd: agent.DeployCommandRestart(leader, &agent.DeployCommand{Archive: ours, Options: opts}), released: true},
		"cancel without an id":           {cmd: agent.DeployCommandCancel("tester"), released: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			d := newDeployment(simpeers{local: leader})
			require.NoError(t, d.Decode(ctx, begin(ours)))

			require.NoError(t, d.Decode(ctx, agent.NewDeployCommand(leader, tc.cmd)))

			err := d.Decode(ctx, begin(other))
			if tc.released {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, "deploy already in progress")
			}
		})
	}
}

func TestDeploymentStaleDoneKeepsRunningDeploy(t *testing.T) {
	var (
		leader = agent.NewPeer("leader")
		ours   = &agent.Archive{DeploymentID: []byte("ours")}
		other  = &agent.Archive{DeploymentID: []byte("other")}
		opts   = &agent.DeployOptions{}
		ctx    = TranscoderContext{State: StateHealthy}
		d      = newDeployment(simpeers{local: leader})
	)

	require.NoError(t, d.Decode(ctx, agent.NewDeployCommand(leader, agent.DeployCommandBegin("tester", ours, opts))))
	require.NoError(t, d.Decode(ctx, agent.NewDeployCommand(leader, agent.DeployCommandDone("tester", other.DeployOption, opts.DeployOption))))

	require.Equal(t, ours.DeploymentID, d.getRunningDeploy().GetArchive().GetDeploymentID())
	require.Equal(t, agent.InfoResponse_Deploying, d.getInfo(nil, nil).Mode)
}
