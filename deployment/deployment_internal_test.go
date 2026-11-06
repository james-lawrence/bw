package deployment

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

// collector is a dispatcher that records every message.
type collector struct {
	m        sync.Mutex
	messages []*agent.Message
}

func (t *collector) Dispatch(_ context.Context, messages ...*agent.Message) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.messages = append(t.messages, messages...)
	return nil
}

// the quorum and the clients attribute a deploy to whoever initiated it, so every deploy event must carry it.
func TestDeployResultCarriesTheInitiator(t *testing.T) {
	for name, tc := range map[string]struct {
		failure error
		stage   agent.Deploy_Stage
	}{
		"completed": {stage: agent.Deploy_Completed},
		"failed":    {failure: errors.New("boom"), stage: agent.Deploy_Failed},
	} {
		t.Run(name, func(t *testing.T) {
			c := &collector{}
			r := DeployResult{
				DeployContext: DeployContext{
					Local:         agent.NewPeer("node"),
					Initiator:     "alice",
					Archive:       &agent.Archive{DeploymentID: []byte("d1")},
					DeployOptions: &agent.DeployOptions{},
					Log:           StdErrLogger(""),
					dispatcher:    c,
					deadline:      t.Context(),
				},
				Error: tc.failure,
			}

			result := r.complete()
			require.Equal(t, tc.stage, result.Stage)
			require.Equal(t, "alice", result.Initiator)

			var events []*agent.Deploy
			for _, m := range c.messages {
				if d := m.GetDeploy(); d != nil {
					events = append(events, d)
				}
			}

			require.Len(t, events, 1)
			require.Equal(t, tc.stage, events[0].Stage)
			require.Equal(t, "alice", events[0].Initiator)
		})
	}
}
