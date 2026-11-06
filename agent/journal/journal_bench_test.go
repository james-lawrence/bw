package journal_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"
	"github.com/james-lawrence/bw/internal/testingx"
)

// BenchmarkJournalFollowers every watcher of the quorum (a deploying client, the coordinator, actl watch)
// follows the journal. each follower lists the journal directory and reopens the segment on every wake while the
// writer appends a steady stream of messages, one every millisecond. the reads of the followers show up as read
// syscalls and bytes per message written, and should stay flat as followers are added.
func BenchmarkJournalFollowers(b *testing.B) {
	for _, followers := range []int{0, 1, 10, 50} {
		b.Run(fmt.Sprintf("followers-%d", followers), func(b *testing.B) {
			j, err := journal.New(testingx.TempDir(b))
			if err != nil {
				b.Fatal(err)
			}
			defer j.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var (
				ready sync.WaitGroup
				done  sync.WaitGroup
			)

			for i := 0; i < followers; i++ {
				ready.Add(1)
				done.Add(1)

				go func() {
					defer done.Done()

					seq := j.Connect(agent.Cursor{})
					ready.Done()

					seen := 0
					for range seq.Each(ctx) {
						if seen++; seen == b.N {
							return
						}
					}
				}()
			}
			ready.Wait()

			peer := agent.NewPeer("bench")
			report := testingx.BenchIO(b)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m := agent.LogInfo(peer, "a typical progress message from a node deploying")
				agent.Stamp(m)
				if err = j.Dispatch(ctx, m); err != nil {
					b.Fatal(err)
				}
				time.Sleep(time.Millisecond)
			}
			done.Wait()
			b.StopTimer()
			report(b.N)
		})
	}
}
