package journal_test

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"
	"github.com/james-lawrence/bw/internal/iterx"

	"github.com/stretchr/testify/require"
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

func segments(t *testing.T, dir string) []string {
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
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

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func receive(t *testing.T, c <-chan *agent.Message, n int) []string {
	t.Helper()
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		select {
		case m := <-c:
			out = append(out, m.GetLog().GetLog())
		case <-time.After(2 * time.Second):
			require.FailNow(t, "timed out waiting for a message")
		}
	}
	return out
}

// requireNoReceive asserts nothing is received on the channel for the duration.
func requireNoReceive[T any](t *testing.T, c <-chan T, d time.Duration) {
	t.Helper()
	select {
	case <-c:
		require.FailNow(t, "unexpected receive")
	case <-time.After(d):
	}
}

// requireReceive waits for a value from the channel.
func requireReceive[T any](t *testing.T, c <-chan T, d time.Duration) T {
	t.Helper()
	select {
	case v := <-c:
		return v
	case <-time.After(d):
		require.FailNow(t, "timed out waiting to receive")
	}
	panic("unreachable")
}

func newJournal(t *testing.T) (string, *journal.Journal) {
	t.Helper()
	dir := t.TempDir()
	j, err := journal.New(dir, journal.OptionDebounce(20*time.Millisecond))
	require.NoError(t, err)
	t.Cleanup(func() { j.Close() })
	return dir, j
}

func TestJournalSegments(t *testing.T) {
	t.Run("should place v4 messages in the window of their timestamp", func(t *testing.T) {
		dir, j := newJournal(t)
		require.NoError(t, j.Dispatch(context.Background(), v4("a", epoch.Add(time.Second)), v4("b", epoch.Add(16*time.Second))))
		require.ElementsMatch(t, []string{
			journal.Segment(epoch),
			journal.Segment(epoch.Add(15 * time.Second)),
		}, segments(t, dir))
	})

	t.Run("should place v7 messages in the window of their id ignoring the timestamp", func(t *testing.T) {
		dir, j := newJournal(t)
		require.NoError(t, j.Dispatch(context.Background(), v7("a", epoch.Add(time.Second))))
		require.ElementsMatch(t, []string{journal.Segment(epoch)}, segments(t, dir))
	})

	t.Run("should name segments so that lexicographic order is time order", func(t *testing.T) {
		require.True(t, journal.Segment(epoch) < journal.Segment(epoch.Add(15*time.Second)))
	})

	t.Run("should reject messages without a timestamp", func(t *testing.T) {
		dir, j := newJournal(t)
		m := v4("untimed", epoch)
		m.Ts = 0
		require.Error(t, j.Dispatch(context.Background(), m))
		require.Empty(t, segments(t, dir))
	})

	t.Run("should reject dispatches once closed", func(t *testing.T) {
		_, j := newJournal(t)
		require.NoError(t, j.Close())
		require.NoError(t, j.Close())
		require.Error(t, j.Dispatch(context.Background(), v4("late", epoch)))
	})

	t.Run("should start empty, discarding what a previous run left in the directory", func(t *testing.T) {
		dir, j := newJournal(t)
		require.NoError(t, j.Dispatch(context.Background(), v4("1", epoch), v4("2", epoch.Add(30*time.Second))))
		require.NoError(t, j.Close())
		require.Len(t, segments(t, dir), 2)

		restarted, err := journal.New(dir)
		require.NoError(t, err)
		t.Cleanup(func() { restarted.Close() })

		require.Empty(t, restarted.Segments())
		require.Empty(t, segments(t, dir))
	})

	t.Run("should journal hidden messages", func(t *testing.T) {
		dir, j := newJournal(t)
		m := v4("hidden", epoch)
		m.Hidden = true
		require.NoError(t, j.Dispatch(context.Background(), m))
		require.Len(t, segments(t, dir), 1)
	})

	t.Run("should prune segments older than the retention", func(t *testing.T) {
		j, err := journal.New(t.TempDir(), journal.OptionRetention(time.Minute))
		require.NoError(t, err)
		defer j.Close()
		now := time.Now()
		require.NoError(t, j.Dispatch(context.Background(), v4("old", now.Add(-time.Hour))))
		require.NoError(t, j.Dispatch(context.Background(), v4("new", now)))
		require.Eventually(t, func() bool {
			s := j.Segments()
			return len(s) == 1 && s[0] == journal.Segment(now)
		}, time.Second, 10*time.Millisecond)
	})
}

func TestJournalTail(t *testing.T) {
	t.Run("is the zero cursor for an empty journal", func(t *testing.T) {
		_, j := newJournal(t)
		require.Equal(t, agent.Cursor{}, j.Tail())
	})

	t.Run("is one past the last message written to the newest segment", func(t *testing.T) {
		dir, j := newJournal(t)
		require.NoError(t, j.Dispatch(context.Background(), v4("1", epoch), v4("2", epoch.Add(time.Second)), v4("3", epoch.Add(20*time.Second))))

		newest := journal.Segment(epoch.Add(20 * time.Second))
		info, err := os.Stat(filepath.Join(dir, newest))
		require.NoError(t, err)

		tail := j.Tail()
		require.Equal(t, agent.WindowCursor(epoch.Add(20*time.Second)).Segment(), tail.Segment())
		require.Equal(t, info.Size(), tail.Offset())

		require.NoError(t, j.Dispatch(context.Background(), v4("4", epoch.Add(21*time.Second))))
		info, err = os.Stat(filepath.Join(dir, newest))
		require.NoError(t, err)
		require.Equal(t, info.Size(), j.Tail().Offset())
	})

	t.Run("does not move for a late message of an older window", func(t *testing.T) {
		_, j := newJournal(t)
		require.NoError(t, j.Dispatch(context.Background(), v4("new", epoch.Add(20*time.Second))))
		before := j.Tail()
		require.NoError(t, j.Dispatch(context.Background(), v4("late", epoch)))
		require.Equal(t, before, j.Tail())
	})
}

func TestJournalFollow(t *testing.T) {
	// setup creates a journal with a cancellable context and an output channel.
	setup := func(t *testing.T) (context.Context, context.CancelFunc, *journal.Journal, chan *agent.Message) {
		t.Helper()
		_, j := newJournal(t)
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		return ctx, cancel, j, make(chan *agent.Message, 32)
	}

	t.Run("should replay from the beginning without duplicates and then tail", func(t *testing.T) {
		ctx, _, j, out := setup(t)
		require.NoError(t, j.Dispatch(ctx, v4("1", epoch), v4("2", epoch.Add(20*time.Second))))
		go pump(ctx, j.Connect(agent.WindowCursor(epoch), true), out)
		require.Equal(t, []string{"1", "2"}, receive(t, out, 2))
		require.NoError(t, j.Dispatch(ctx, v4("3", epoch.Add(21*time.Second)), v4("4", epoch.Add(40*time.Second))))
		require.Equal(t, []string{"3", "4"}, receive(t, out, 2))
		requireNoReceive(t, out, 200*time.Millisecond)
	})

	t.Run("should start at the tail by default", func(t *testing.T) {
		ctx, _, j, out := setup(t)
		require.NoError(t, j.Dispatch(ctx, v4("old", epoch)))
		go pump(ctx, j.Connect(agent.Cursor{}, true), out)
		requireNoReceive(t, out, 200*time.Millisecond)
		require.NoError(t, j.Dispatch(ctx, v4("new", epoch.Add(time.Second))))
		require.Equal(t, []string{"new"}, receive(t, out, 1))
	})

	t.Run("should resume from a cursor", func(t *testing.T) {
		ctx, _, j, out := setup(t)
		require.NoError(t, j.Dispatch(ctx, v4("1", epoch), v4("2", epoch)))
		cursor := j.Tail()
		require.NoError(t, j.Dispatch(ctx, v4("3", epoch), v4("4", epoch.Add(15*time.Second))))
		go pump(ctx, j.Connect(cursor, true), out)
		require.Equal(t, []string{"3", "4"}, receive(t, out, 2))
	})

	t.Run("should replay the window of a message when resumed from its cursor", func(t *testing.T) {
		ctx, _, j, out := setup(t)
		m1, m2, m3 := v7("1", epoch), v7("2", epoch.Add(time.Second)), v7("3", epoch.Add(20*time.Second))
		require.NoError(t, j.Dispatch(ctx, m1, m2, m3))
		go pump(ctx, j.Connect(agent.CursorOf(m2), true), out)
		require.Equal(t, []string{"1", "2", "3"}, receive(t, out, 3))
	})

	t.Run("should skip windows that have been pruned", func(t *testing.T) {
		ctx, _, j, out := setup(t)
		require.NoError(t, j.Dispatch(ctx, v4("1", epoch.Add(30*time.Second))))
		go pump(ctx, j.Connect(agent.WindowCursor(epoch), true), out)
		require.Equal(t, []string{"1"}, receive(t, out, 1))
	})

	t.Run("should not deliver hidden messages", func(t *testing.T) {
		ctx, _, j, out := setup(t)
		hidden := v4("hidden", epoch)
		hidden.Hidden = true
		go pump(ctx, j.Connect(agent.Cursor{}, true), out)
		require.NoError(t, j.Dispatch(ctx, hidden, v4("visible", epoch)))
		require.Equal(t, []string{"visible"}, receive(t, out, 1))
	})

	t.Run("should not block the writer when a reader is stalled", func(t *testing.T) {
		ctx, _, j, _ := setup(t)
		stalled := make(chan *agent.Message) // never read.
		go pump(ctx, j.Connect(agent.Cursor{}, true), stalled)

		start := time.Now()
		for i := 0; i < 1000; i++ {
			require.NoError(t, j.Dispatch(ctx, v4("x", epoch)))
		}
		require.Less(t, time.Since(start), time.Second)
	})

	t.Run("should stop when the context is cancelled", func(t *testing.T) {
		ctx, cancel, j, _ := setup(t)
		seq := j.Connect(agent.Cursor{}, true)
		done := make(chan error, 1)
		go func() {
			for range seq.Each(ctx) {
			}
			done <- seq.Err()
		}()
		cancel()
		require.ErrorIs(t, requireReceive(t, done, time.Second), context.Canceled)
	})

	t.Run("should stop without an error when the consumer stops", func(t *testing.T) {
		ctx, _, j, _ := setup(t)
		require.NoError(t, j.Dispatch(ctx, v4("1", epoch), v4("2", epoch), v4("3", epoch)))
		seq := iterx.N(2, j.Connect(agent.WindowCursor(epoch), true))

		var got []string
		for m := range seq.Each(ctx) {
			got = append(got, m.GetLog().GetLog())
		}

		require.Equal(t, []string{"1", "2"}, got)
		require.NoError(t, seq.Err())
	})

	t.Run("should release a follower that is caught up and waiting when the context is cancelled", func(t *testing.T) {
		ctx, cancel, j, _ := setup(t)
		require.NoError(t, j.Dispatch(ctx, v4("1", epoch)))

		seq := j.Connect(agent.WindowCursor(epoch), true)
		done := make(chan error, 1)
		go func() {
			for range seq.Each(ctx) {
				// delivered, now caught up and parked waiting for more.
			}
			done <- seq.Err()
		}()

		requireNoReceive(t, done, 100*time.Millisecond)
		cancel()
		require.ErrorIs(t, requireReceive(t, done, time.Second), context.Canceled)
	})

	t.Run("should end a waiting follower with an error when the journal is closed", func(t *testing.T) {
		ctx, _, j, _ := setup(t)
		seq := j.Connect(agent.Cursor{}, true)
		done := make(chan error, 1)
		go func() {
			for range seq.Each(ctx) {
			}
			done <- seq.Err()
		}()

		requireNoReceive(t, done, 100*time.Millisecond)
		require.NoError(t, j.Close())
		require.ErrorContains(t, requireReceive(t, done, time.Second), "journal is closed")
	})

	t.Run("should deliver a burst of writes completely and in order", func(t *testing.T) {
		ctx, _, _, out := setup(t)
		burst, err := journal.New(t.TempDir(), journal.OptionDebounce(100*time.Millisecond))
		require.NoError(t, err)
		t.Cleanup(func() { burst.Close() })

		go pump(ctx, burst.Connect(agent.Cursor{}, true), out)
		time.Sleep(50 * time.Millisecond) // the follower is parked.

		expected := make([]string, 0, 20)
		for i := 0; i < 20; i++ {
			expected = append(expected, strconv.Itoa(i))
			require.NoError(t, burst.Dispatch(ctx, v4(strconv.Itoa(i), epoch)))
		}

		require.Equal(t, expected, receive(t, out, 20))
	})

	t.Run("should not starve a follower while writes keep arriving", func(t *testing.T) {
		ctx, _, _, out := setup(t)
		steady, err := journal.New(t.TempDir(), journal.OptionDebounce(100*time.Millisecond))
		require.NoError(t, err)
		t.Cleanup(func() { steady.Close() })

		go pump(ctx, steady.Connect(agent.Cursor{}, true), out)

		// a write every 10ms for 600ms, a debounce that restarts on every write would deliver nothing until it ends.
		start := time.Now()
		go func() {
			for i := 0; i < 60 && ctx.Err() == nil; i++ {
				_ = steady.Dispatch(ctx, v4(strconv.Itoa(i), epoch))
				time.Sleep(10 * time.Millisecond)
			}
		}()

		requireReceive(t, out, 400*time.Millisecond)
		require.Less(t, time.Since(start), 450*time.Millisecond)
	})
}
