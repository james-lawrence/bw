package ux

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"github.com/gizak/termui"
)

// NewTermui - terminal based ux.
func NewTermui(ctx context.Context, wg *sync.WaitGroup, events chan agent.Message) {
	wg.Add(1)
	defer wg.Done()

	var (
		storage = state{
			Peers: map[agent.Peer]deployment.Status{},
		}
	)

	defer termui.Close()
	defer termui.Clear()

	log.Println("termui started")
	if err := termui.Init(); err != nil {
		log.Println("failed to initialized ui", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case m := <-events:
			storage = mergeEvent(storage, m)
		}

		render(storage)
	}
}

func mergeEvent(s state, m agent.Message) state {
	switch m.Type {
	// case agent.Message_PeerEvent:

	case agent.Message_DeployEvent:
	case agent.Message_PeersCompletedEvent:
		s.NodesCompleted = m.GetInt()
	case agent.Message_PeersFoundEvent:
		s.NodesFound = m.GetInt()
	default:
		log.Printf("%s - %s: \n", time.Unix(m.GetTs(), 0).Format(time.Stamp), m.Type)
	}

	return s
}

func render(s state) {
	termWidth := termui.TermWidth()
	termHeight := termui.TermHeight()
	completed := termui.NewGauge()
	completed.Height = 5
	completed.Width = termWidth
	completed.Percent = int((float64(s.NodesCompleted) / float64(s.NodesFound)) * 100)
	peers := termui.NewList()
	peers.Items = peersToList(s)
	peers.Width = termWidth
	peers.Height = termHeight - 10

	g := termui.NewGrid(
		termui.NewRow(
			termui.NewCol(6, 0, completed),
		),
		termui.NewRow(
			termui.NewCol(6, 0, peers),
		),
	)
	g.Width = termWidth

	g.Align()
	termui.Clear()
	termui.Render(g)
}

func peersToList(s state) []string {
	peers := make([]agent.Peer, 0, len(s.Peers))
	for peer := range s.Peers {
		peers = append(peers, peer)
	}
	sort.Sort(sortablePeers(peers))

	result := make([]string, 0, len(s.Peers))
	for _, peer := range peers {
		result = append(result, fmt.Sprintf("%s: %s", peer.Name, s.Peers[peer]))
	}
	return result
}

type state struct {
	NodesFound     int64
	NodesCompleted int64
	Peers          map[agent.Peer]deployment.Status
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
	return strings.Compare(t[i].Ip, t[j].Ip) == -1
}
