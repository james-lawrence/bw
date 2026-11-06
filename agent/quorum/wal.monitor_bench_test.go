package quorum

import (
	"context"
	"io"
	"log"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
)

type countingDispatcher struct{ n atomic.Int64 }

func (t *countingDispatcher) Dispatch(_ context.Context, m ...*agent.Message) error {
	t.n.Add(int64(len(m)))
	return nil
}

// BenchmarkApplyMonitorSlow the apply monitor reports back through raft, so a stalled state machine could amplify
// itself by adding entries. every apply in the deploy is slow, the reports should stay constant per deploy: the first
// slow apply, and the maximum at the end.
func BenchmarkApplyMonitorSlow(b *testing.B) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(log.Writer())

	peer := agent.NewPeer("bench")

	for _, applies := range []int{10, 1000, 100000} {
		b.Run(applyName(applies), func(b *testing.B) {
			d := &countingDispatcher{}
			m := newApplyMonitor(d, func() *agent.Peer { return peer })

			begin := agent.NewDeployCommand(peer, agent.DeployCommandBegin("bench", &agent.Archive{}, &agent.DeployOptions{}))
			done := agent.NewDeployCommand(peer, agent.DeployCommandDone("bench"))
			l := &raft.Log{}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m.begin(begin)
				for j := 0; j < applies; j++ {
					m.observe(l, 5*time.Second)
				}
				m.end(done)
			}
			b.StopTimer()

			if err := m.Close(); err != nil {
				b.Fatal(err)
			}

			b.ReportMetric(float64(d.n.Load())/float64(b.N), "reports/deploy")
		})
	}
}

func applyName(n int) string {
	switch n {
	case 10:
		return "applies-10"
	case 1000:
		return "applies-1000"
	default:
		return "applies-100000"
	}
}
