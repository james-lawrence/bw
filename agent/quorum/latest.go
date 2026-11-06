package quorum

import (
	"context"
	"slices"
	"strings"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/errorsx"
)

// historyLimit the number of messages of the latest deploy that are kept.
const historyLimit = 128

// latest is the state of the deployments recovered from the journal.
type latest struct {
	// history the messages of the latest deploy, without heartbeats.
	history []*agent.Message
	// deployed the latest completed deploy event of each node, by node name.
	deployed map[string]*agent.Message
}

// completed the completed deploy events of the nodes, in the order of the node names.
func (t latest) completed() []*agent.Message {
	events := make([]*agent.Message, 0, len(t.deployed))
	for _, m := range t.deployed {
		events = append(events, m)
	}

	slices.SortFunc(events, func(a, b *agent.Message) int { return strings.Compare(a.GetPeer().GetName(), b.GetPeer().GetName()) })

	return events
}

// scanLatest reads the journal and derives the history of the latest deploy, and the deployment each node completed.
// the journal holds the messages since the last successful deploy began, which is what both need.
func scanLatest(ctx context.Context, d agent.ConnectableDispatcher) (_ latest, err error) {
	var (
		seen   = make(map[string]struct{}, 256)
		result = latest{deployed: make(map[string]*agent.Message, 128)}
	)

	seq := d.Connect(agent.Cursor{}, false)
	for m := range seq.Each(ctx) {
		// restoring a snapshot replays messages the journal may already hold.
		if _, dup := seen[m.GetId()]; dup && m.GetId() != "" {
			continue
		}
		seen[m.GetId()] = struct{}{}

		result.observeDeployed(m)
		result.observeHistory(m)
	}

	return result, errorsx.Compact(seq.Err(), ctx.Err())
}

func (t *latest) observeHistory(m *agent.Message) {
	switch m.GetType() {
	case agent.Message_DeployHeartbeat, agent.Message_RestoreEventInitiated, agent.Message_RestoreEventCompleted:
		// heartbeats only signal liveness, recording them evicts the actual history. the markers are local.
		return
	case agent.Message_DeployCommandEvent:
		// the history is of the latest deploy.
		if m.GetDeployCommand().GetCommand() == agent.DeployCommand_Begin {
			t.history = t.history[:0]
		}
	}

	t.history = append(t.history, m)
	if len(t.history) > historyLimit {
		t.history = append(t.history[:0], t.history[len(t.history)-historyLimit:]...)
	}
}

// observeDeployed records the deployments nodes complete, and forgets the nodes that depart. a deploy that isn't
// complete leaves what the node had.
func (t *latest) observeDeployed(m *agent.Message) {
	p := m.GetPeer()
	if p == nil {
		return
	}

	switch event := m.GetEvent().(type) {
	case *agent.Message_Deploy:
		if event.Deploy.GetStage() == agent.Deploy_Completed && len(event.Deploy.GetArchive().GetDeploymentID()) > 0 {
			t.deployed[p.Name] = m
		}
	case *agent.Message_Membership:
		if event.Membership == agent.Message_Departed {
			delete(t.deployed, p.Name)
		}
	}
}
