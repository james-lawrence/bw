package journal_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"
	"github.com/james-lawrence/bw/internal/iterx"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var epoch = time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC)

// message with a v4 id, the window is determined by Ts.
func v4(text string, ts time.Time) *agent.Message {
	m := agent.LogInfo(agent.NewPeer("node"), text)
	m.Id = uuid.Must(uuid.NewV4()).String()
	m.Ts = ts.Unix()
	return m
}

// message with a v7 id, the window is determined by the id, Ts is ignored.
func v7(text string, ts time.Time) *agent.Message {
	m := agent.LogInfo(agent.NewPeer("node"), text)
	m.Id = uuid.Must(uuid.NewV7AtTime(ts)).String()
	m.Ts = ts.Add(24 * time.Hour).UnixNano()
	return m
}

func segments(dir string) []string {
	entries, err := os.ReadDir(dir)
	Expect(err).To(Succeed())
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// pump the sequence into the channel until the context is cancelled, returns the error of the sequence.
func pump(ctx context.Context, seq iterx.Seq[*agent.Message], out chan<- *agent.Message) error {
	for m := range seq.Each(ctx) {
		select {
		case out <- m:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return seq.Err()
}

func receive(c <-chan *agent.Message, n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		var m *agent.Message
		Eventually(c, 2*time.Second).Should(Receive(&m))
		out = append(out, m.GetLog().GetLog())
	}
	return out
}

var _ = Describe("Journal", func() {
	var (
		dir string
		j   *journal.Journal
	)

	BeforeEach(func() {
		var err error
		dir = GinkgoT().TempDir()
		j, err = journal.New(dir, journal.OptionDebounce(20*time.Millisecond))
		Expect(err).To(Succeed())
		DeferCleanup(j.Close)
	})

	Describe("segments", func() {
		It("should place v4 messages in the window of their timestamp", func() {
			Expect(j.Dispatch(context.Background(), v4("a", epoch.Add(time.Second)), v4("b", epoch.Add(16*time.Second)))).To(Succeed())
			Expect(segments(dir)).To(ConsistOf(
				journal.Segment(epoch),
				journal.Segment(epoch.Add(15*time.Second)),
			))
		})

		It("should place v7 messages in the window of their id ignoring the timestamp", func() {
			Expect(j.Dispatch(context.Background(), v7("a", epoch.Add(time.Second)))).To(Succeed())
			Expect(segments(dir)).To(ConsistOf(journal.Segment(epoch)))
		})

		It("should name segments so that lexicographic order is time order", func() {
			Expect(journal.Segment(epoch) < journal.Segment(epoch.Add(15*time.Second))).To(BeTrue())
		})

		It("should reject messages without a timestamp", func() {
			m := v4("untimed", epoch)
			m.Ts = 0
			Expect(j.Dispatch(context.Background(), m)).ToNot(Succeed())
			Expect(segments(dir)).To(BeEmpty())
		})

		It("should reject dispatches once closed", func() {
			Expect(j.Close()).To(Succeed())
			Expect(j.Close()).To(Succeed())
			Expect(j.Dispatch(context.Background(), v4("late", epoch))).ToNot(Succeed())
		})

		It("should start empty, discarding what a previous run left in the directory", func() {
			Expect(j.Dispatch(context.Background(), v4("1", epoch), v4("2", epoch.Add(30*time.Second)))).To(Succeed())
			Expect(j.Close()).To(Succeed())
			Expect(segments(dir)).To(HaveLen(2))

			restarted, err := journal.New(dir)
			Expect(err).To(Succeed())
			DeferCleanup(restarted.Close)

			Expect(restarted.Segments()).To(BeEmpty())
			Expect(segments(dir)).To(BeEmpty())
		})

		It("should journal hidden messages", func() {
			m := v4("hidden", epoch)
			m.Hidden = true
			Expect(j.Dispatch(context.Background(), m)).To(Succeed())
			Expect(segments(dir)).To(HaveLen(1))
		})

		It("should prune segments older than the retention", func() {
			j, err := journal.New(GinkgoT().TempDir(), journal.OptionRetention(time.Minute))
			Expect(err).To(Succeed())
			defer j.Close()
			now := time.Now()
			Expect(j.Dispatch(context.Background(), v4("old", now.Add(-time.Hour)))).To(Succeed())
			Expect(j.Dispatch(context.Background(), v4("new", now))).To(Succeed())
			Eventually(j.Segments).Should(Equal([]string{journal.Segment(now)}))
		})
	})

	Describe("tail", func() {
		It("is the zero cursor for an empty journal", func() {
			Expect(j.Tail()).To(Equal(agent.Cursor{}))
		})

		It("is one past the last message written to the newest segment", func() {
			Expect(j.Dispatch(context.Background(), v4("1", epoch), v4("2", epoch.Add(time.Second)), v4("3", epoch.Add(20*time.Second)))).To(Succeed())

			newest := journal.Segment(epoch.Add(20 * time.Second))
			info, err := os.Stat(filepath.Join(dir, newest))
			Expect(err).To(Succeed())

			tail := j.Tail()
			Expect(tail.Segment()).To(Equal(agent.WindowCursor(epoch.Add(20 * time.Second)).Segment()))
			Expect(tail.Offset()).To(Equal(info.Size()))

			Expect(j.Dispatch(context.Background(), v4("4", epoch.Add(21*time.Second)))).To(Succeed())
			info, err = os.Stat(filepath.Join(dir, newest))
			Expect(err).To(Succeed())
			Expect(j.Tail().Offset()).To(Equal(info.Size()))
		})

		It("does not move for a late message of an older window", func() {
			Expect(j.Dispatch(context.Background(), v4("new", epoch.Add(20*time.Second)))).To(Succeed())
			before := j.Tail()
			Expect(j.Dispatch(context.Background(), v4("late", epoch))).To(Succeed())
			Expect(j.Tail()).To(Equal(before))
		})
	})

	Describe("follow", func() {
		var (
			ctx    context.Context
			cancel context.CancelFunc
			out    chan *agent.Message
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			DeferCleanup(func() { cancel() })
			out = make(chan *agent.Message, 32)
		})

		It("should replay from the beginning without duplicates and then tail", func() {
			Expect(j.Dispatch(ctx, v4("1", epoch), v4("2", epoch.Add(20*time.Second)))).To(Succeed())
			go pump(ctx, j.Connect(agent.WindowCursor(epoch)), out)
			Expect(receive(out, 2)).To(Equal([]string{"1", "2"}))
			Expect(j.Dispatch(ctx, v4("3", epoch.Add(21*time.Second)), v4("4", epoch.Add(40*time.Second)))).To(Succeed())
			Expect(receive(out, 2)).To(Equal([]string{"3", "4"}))
			Consistently(out, 200*time.Millisecond).ShouldNot(Receive())
		})

		It("should start at the tail by default", func() {
			Expect(j.Dispatch(ctx, v4("old", epoch))).To(Succeed())
			go pump(ctx, j.Connect(agent.Cursor{}), out)
			Consistently(out, 200*time.Millisecond).ShouldNot(Receive())
			Expect(j.Dispatch(ctx, v4("new", epoch.Add(time.Second)))).To(Succeed())
			Expect(receive(out, 1)).To(Equal([]string{"new"}))
		})

		It("should resume from a cursor", func() {
			Expect(j.Dispatch(ctx, v4("1", epoch), v4("2", epoch))).To(Succeed())
			cursor := j.Tail()
			Expect(j.Dispatch(ctx, v4("3", epoch), v4("4", epoch.Add(15*time.Second)))).To(Succeed())
			go pump(ctx, j.Connect(cursor), out)
			Expect(receive(out, 2)).To(Equal([]string{"3", "4"}))
		})

		It("should replay the window of a message when resumed from its cursor", func() {
			m1, m2, m3 := v7("1", epoch), v7("2", epoch.Add(time.Second)), v7("3", epoch.Add(20*time.Second))
			Expect(j.Dispatch(ctx, m1, m2, m3)).To(Succeed())
			go pump(ctx, j.Connect(agent.CursorOf(m2)), out)
			Expect(receive(out, 3)).To(Equal([]string{"1", "2", "3"}))
		})

		It("should skip windows that have been pruned", func() {
			Expect(j.Dispatch(ctx, v4("1", epoch.Add(30*time.Second)))).To(Succeed())
			go pump(ctx, j.Connect(agent.WindowCursor(epoch)), out)
			Expect(receive(out, 1)).To(Equal([]string{"1"}))
		})

		It("should not deliver hidden messages", func() {
			hidden := v4("hidden", epoch)
			hidden.Hidden = true
			go pump(ctx, j.Connect(agent.Cursor{}), out)
			Expect(j.Dispatch(ctx, hidden, v4("visible", epoch))).To(Succeed())
			Expect(receive(out, 1)).To(Equal([]string{"visible"}))
		})

		It("should not block the writer when a reader is stalled", func() {
			stalled := make(chan *agent.Message) // never read.
			go pump(ctx, j.Connect(agent.Cursor{}), stalled)

			start := time.Now()
			for i := 0; i < 1000; i++ {
				Expect(j.Dispatch(ctx, v4("x", epoch))).To(Succeed())
			}
			Expect(time.Since(start)).To(BeNumerically("<", time.Second))
		})

		It("should stop when the context is cancelled", func() {
			seq := j.Connect(agent.Cursor{})
			done := make(chan error, 1)
			go func() {
				for range seq.Each(ctx) {
				}
				done <- seq.Err()
			}()
			cancel()
			Eventually(done, time.Second).Should(Receive(MatchError(context.Canceled)))
		})

		It("should stop without an error when the consumer stops", func() {
			Expect(j.Dispatch(ctx, v4("1", epoch), v4("2", epoch), v4("3", epoch))).To(Succeed())
			seq := iterx.N(2, j.Connect(agent.WindowCursor(epoch)))

			var got []string
			for m := range seq.Each(ctx) {
				got = append(got, m.GetLog().GetLog())
			}

			Expect(got).To(Equal([]string{"1", "2"}))
			Expect(seq.Err()).To(Succeed())
		})

		It("should release a follower that is caught up and waiting when the context is cancelled", func() {
			Expect(j.Dispatch(ctx, v4("1", epoch))).To(Succeed())

			seq := j.Connect(agent.WindowCursor(epoch))
			done := make(chan error, 1)
			go func() {
				for range seq.Each(ctx) {
					// delivered, now caught up and parked waiting for more.
				}
				done <- seq.Err()
			}()

			Consistently(done, 100*time.Millisecond).ShouldNot(Receive())
			cancel()
			Eventually(done, time.Second).Should(Receive(MatchError(context.Canceled)))
		})

		It("should end a waiting follower with an error when the journal is closed", func() {
			seq := j.Connect(agent.Cursor{})
			done := make(chan error, 1)
			go func() {
				for range seq.Each(ctx) {
				}
				done <- seq.Err()
			}()

			Consistently(done, 100*time.Millisecond).ShouldNot(Receive())
			Expect(j.Close()).To(Succeed())
			Eventually(done, time.Second).Should(Receive(MatchError(ContainSubstring("journal is closed"))))
		})

		It("should deliver a burst of writes completely and in order", func() {
			burst, err := journal.New(GinkgoT().TempDir(), journal.OptionDebounce(100*time.Millisecond))
			Expect(err).To(Succeed())
			DeferCleanup(burst.Close)

			go pump(ctx, burst.Connect(agent.Cursor{}), out)
			time.Sleep(50 * time.Millisecond) // the follower is parked.

			expected := make([]string, 0, 20)
			for i := 0; i < 20; i++ {
				expected = append(expected, strconv.Itoa(i))
				Expect(burst.Dispatch(ctx, v4(strconv.Itoa(i), epoch))).To(Succeed())
			}

			Expect(receive(out, 20)).To(Equal(expected))
		})

		It("should not starve a follower while writes keep arriving", func() {
			steady, err := journal.New(GinkgoT().TempDir(), journal.OptionDebounce(100*time.Millisecond))
			Expect(err).To(Succeed())
			DeferCleanup(steady.Close)

			go pump(ctx, steady.Connect(agent.Cursor{}), out)

			// a write every 10ms for 600ms, a debounce that restarts on every write would deliver nothing until it ends.
			start := time.Now()
			go func() {
				for i := 0; i < 60 && ctx.Err() == nil; i++ {
					_ = steady.Dispatch(ctx, v4(strconv.Itoa(i), epoch))
					time.Sleep(10 * time.Millisecond)
				}
			}()

			Eventually(out, 400*time.Millisecond).Should(Receive())
			Expect(time.Since(start)).To(BeNumerically("<", 450*time.Millisecond))
		})
	})
})
