package ux

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/gizak/termui"
)

// NewTermui - terminal based ux.
func NewTermui(ctx context.Context, done context.CancelFunc, wg *sync.WaitGroup, events chan agent.Message) {
	defer wg.Done()
	defer log.Println("termui shutting down")

	var (
		storage = state{
			Peers: map[string]agent.Peer{},
			Logs:  newLBuffer(300),
		}
	)

	defer termui.Close()
	defer termui.Clear()

	if err := termui.Init(); err != nil {
		log.Println("failed to initialized ui", err)
		return
	}

	termui.Handle("/sys/kbd/C-c", func(e termui.Event) {
		done()
		termui.StopLoop()
	})

	go func() {
		termui.Loop()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-events:
			if !ok {
				return
			}

			storage = mergeEvent(storage, m)
		}

		render(storage)
	}
}

func mergeEvent(s state, m agent.Message) state {
	switch m.Type {
	case agent.Message_LogEvent:
		s.Logs = s.Logs.Add(m)
	case agent.Message_PeerEvent:
		s.Peers[m.Peer.Name] = *m.Peer
	case agent.Message_DeployEvent:
		d := m.GetDeploy()
		s.Logs = s.Logs.Add(agentutil.LogEvent(*m.Peer, fmt.Sprintf("%s - %s %s", m.Type, bw.RandomID(d.Archive.DeploymentID), d.Stage)))
	case agent.Message_PeersCompletedEvent:
		s.NodesCompleted = m.GetInt()
	case agent.Message_PeersFoundEvent:
		s.NodesFound = m.GetInt()
	default:
		s.Logs = s.Logs.Add(agentutil.LogEvent(*m.Peer, fmt.Sprintf("%s - Unknown Event - %s", messagePrefix(m), m.Type)))
	}

	return s
}

func render(s state) {
	termWidth := termui.TermWidth()
	termHeight := termui.TermHeight()
	completed := termui.NewGauge()
	completed.Border = true
	completed.BorderLabel = fmt.Sprintf("progress - %d / %d", s.NodesCompleted, s.NodesFound)
	completed.Height = 5
	completed.Width = termWidth
	if s.NodesFound > 0 {
		completed.Percent = int((float64(s.NodesCompleted) / float64(s.NodesFound)) * 100)
	} else {
		completed.Percent = 0
	}

	agents := termui.NewList()
	agents.Border = true
	agents.BorderLabel = "agents"
	agents.Items = peersToList(s)
	agents.Width = termWidth
	agents.Height = termHeight - completed.Height

	logs := termui.NewList()
	logs.Border = true
	logs.BorderLabel = "info"
	logs.Overflow = "wrap"
	logs.Items = logsToList(s)
	logs.Width = termWidth
	logs.Height = termHeight - completed.Height

	g := termui.NewGrid(
		termui.NewRow(
			termui.NewCol(12, 0, completed),
		),
		termui.NewRow(
			termui.NewCol(2, 0, agents),
			termui.NewCol(10, 0, logs),
		),
	)
	g.Width = termWidth
	g.Align()

	termui.Clear()
	termui.Render(g)
}

func logsToList(s state) []string {
	out := make([]string, 0, s.Logs.ring.Len())
	s.Logs.Do(func(m agent.Message) {
		l := m.GetLog()
		out = append(out, fmt.Sprintf("%s - %s", messagePrefix(m), l.Log))
	})
	return out
}

func peersToList(s state) []string {
	peers := make([]agent.Peer, 0, len(s.Peers))
	for _, peer := range s.Peers {
		peers = append(peers, peer)
	}
	sort.Sort(sortablePeers(peers))

	result := make([]string, 0, len(s.Peers))
	for _, peer := range peers {
		result = append(result, fmt.Sprintf("%s: %s", peer.Name, peer.Status))
	}

	return result
}

type state struct {
	NodesFound     int64
	NodesCompleted int64
	Peers          map[string]agent.Peer
	Logs           lbuffer
}

type sortablePeers []agent.Peer

// Len is part of sort.Interface.
func (t sortablePeers) Len() int {
	return len(t)
}

// Swap is part of sort.Interface.
func (t sortablePeers) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (t sortablePeers) Less(i, j int) bool {
	return strings.Compare(t[i].Name, t[j].Name) == -1
}
