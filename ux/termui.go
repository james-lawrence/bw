package ux

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"bitbucket.org/jatone/bearded-wookie/deployment"
	"github.com/gizak/termui"
	"github.com/hashicorp/memberlist"
)

func NewTermui(wg *sync.WaitGroup, ctx context.Context) *deployment.Events {
	events := deployment.NewEvents()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var (
			storage = state{
				Peers: map[*memberlist.Node]deployment.Status{},
			}
		)

		if err := termui.Init(); err != nil {
			log.Println("failed to initialized ui", err)
			return
		}
		defer termui.Close()
		defer termui.Clear()

		for {
			select {
			case storage.NodesFound = <-events.NodesFound:
			case storage.NodesCompleted = <-events.NodesCompleted:
			case <-ctx.Done():
				return
			case storage.Stage = <-events.StageUpdate:
				if storage.Stage == deployment.StageDone {
					time.Sleep(2 * time.Second)
					return
				}
			case e := <-events.Status:
				if deployment.IsReady(e.Status) {
					delete(storage.Peers, e.Peer)
				} else if s, ok := e.Status.(deployment.Status); ok {
					storage.Peers[e.Peer] = s
				} else {
					// TODO failure
				}
			}

			render(storage)
		}
	}()

	return events
}

func render(s state) {
	termWidth := termui.TermWidth()
	termHeight := termui.TermHeight()
	title := termui.NewPar(stageToString(s))
	title.Width = termWidth
	title.Height = 5
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
			termui.NewCol(6, 0, title),
		),
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
	peers := make([]*memberlist.Node, 0, len(s.Peers))
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

func stageToString(s state) string {
	switch s.Stage {
	case deployment.StageWaitingForReady:
		return "waiting for all nodes to become ready"
	case deployment.StageDeploying:
		return "deploying to nodes"
	case deployment.StageDone:
		return "completed"
	default:
		return ""
	}
}

type state struct {
	NodesFound     int64
	NodesCompleted int64
	Stage          deployment.Stage
	Peers          map[*memberlist.Node]deployment.Status
}

type sortablePeers []*memberlist.Node

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
	return bytes.Compare([]byte(t[i].Addr), []byte(t[j].Addr)) == -1
}
