package journal_test

import (
	"context"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"
	"github.com/stretchr/testify/require"
)

// scan collects the messages of a scan.
func scan(t *testing.T, j *journal.Journal, c agent.Cursor) (logs []string) {
	t.Helper()

	seq := j.Connect(c, false)
	for m := range seq.Each(t.Context()) {
		logs = append(logs, m.GetLog().GetLog())
	}
	require.NoError(t, seq.Err())

	return logs
}

func TestJournalScan(t *testing.T) {
	t.Run("an empty journal has nothing", func(t *testing.T) {
		_, j := newJournal(t)
		require.Empty(t, scan(t, j, agent.Cursor{}))
	})

	t.Run("returns everything written and stops at the end", func(t *testing.T) {
		_, j := newJournal(t)
		require.NoError(t, j.Dispatch(t.Context(),
			v7("a", epoch),
			v7("b", epoch.Add(time.Minute)),
			v7("c", epoch.Add(time.Hour)),
		))

		// returning at all proves it didn't wait for more.
		require.Equal(t, []string{"a", "b", "c"}, scan(t, j, agent.Cursor{}))
	})

	t.Run("starts at the cursor", func(t *testing.T) {
		_, j := newJournal(t)
		require.NoError(t, j.Dispatch(t.Context(),
			v7("a", epoch),
			v7("b", epoch.Add(time.Minute)),
			v7("c", epoch.Add(time.Hour)),
		))

		require.Equal(t, []string{"b", "c"}, scan(t, j, agent.WindowCursor(epoch.Add(time.Minute))))
	})

	t.Run("skips hidden messages", func(t *testing.T) {
		_, j := newJournal(t)
		hidden := v7("hidden", epoch)
		hidden.Hidden = true
		require.NoError(t, j.Dispatch(t.Context(), hidden, v7("shown", epoch.Add(time.Second))))

		require.Equal(t, []string{"shown"}, scan(t, j, agent.Cursor{}))
	})
}

func TestJournalReset(t *testing.T) {
	t.Run("discards the messages", func(t *testing.T) {
		dir, j := newJournal(t)
		require.NoError(t, j.Dispatch(t.Context(), v7("a", epoch), v7("b", epoch.Add(time.Hour))))

		require.NoError(t, j.Reset())
		require.Empty(t, segments(t, dir))
		require.Empty(t, scan(t, j, agent.Cursor{}))
	})

	t.Run("records again afterwards, including older messages", func(t *testing.T) {
		_, j := newJournal(t)
		require.NoError(t, j.Dispatch(t.Context(), v7("a", epoch.Add(time.Hour))))
		require.NoError(t, j.Reset())

		// what a restored snapshot replays is older than what the journal had.
		require.NoError(t, j.Dispatch(t.Context(), v7("b", epoch), v7("c", epoch.Add(time.Minute))))
		require.Equal(t, []string{"b", "c"}, scan(t, j, agent.Cursor{}))
	})

	t.Run("ends the followers", func(t *testing.T) {
		_, j := newJournal(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		out := make(chan *agent.Message, 4)
		result := make(chan error, 1)
		go func() { result <- pump(ctx, j.Connect(agent.Cursor{}, true), out) }()

		// a follower is established once it delivers.
		require.Eventually(t, func() bool {
			require.NoError(t, j.Dispatch(t.Context(), v7("live", time.Now())))
			select {
			case <-out:
				return true
			case <-time.After(100 * time.Millisecond):
				return false
			}
		}, 5*time.Second, 10*time.Millisecond)

		require.NoError(t, j.Reset())

		select {
		case err := <-result:
			require.ErrorIs(t, err, journal.ErrReset)
		case <-time.After(2 * time.Second):
			t.Fatal("the follower kept following a journal that was reset")
		}
	})

	t.Run("followers that connect afterwards are fine", func(t *testing.T) {
		_, j := newJournal(t)
		require.NoError(t, j.Dispatch(t.Context(), v7("a", epoch)))
		require.NoError(t, j.Reset())

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		out := make(chan *agent.Message, 4)
		go func() { _ = pump(ctx, j.Connect(agent.Cursor{}, true), out) }()

		require.Eventually(t, func() bool {
			require.NoError(t, j.Dispatch(t.Context(), v7("after", time.Now())))
			select {
			case m := <-out:
				return m.GetLog().GetLog() == "after"
			case <-time.After(100 * time.Millisecond):
				return false
			}
		}, 5*time.Second, 10*time.Millisecond)
	})

	t.Run("a window cursor from before the reset is safe to resume from", func(t *testing.T) {
		_, j := newJournal(t)
		require.NoError(t, j.Dispatch(t.Context(), v7("old", epoch)))
		cursor := agent.WindowCursor(epoch)
		require.NoError(t, j.Reset())
		require.NoError(t, j.Dispatch(t.Context(), v7("new", epoch.Add(time.Second))))

		require.Equal(t, []string{"new"}, scan(t, j, cursor))
	})
}

func TestJournalRetain(t *testing.T) {
	segmentsAfter := func(t *testing.T, dir string, want int) {
		t.Helper()
		require.Eventually(t, func() bool { return len(segments(t, dir)) == want }, 5*time.Second, 10*time.Millisecond, "%v", segments(t, dir))
	}

	t.Run("keeps a day by default", func(t *testing.T) {
		dir, j := newJournal(t)
		require.NoError(t, j.Dispatch(t.Context(), v7("a", epoch), v7("b", epoch.Add(23*time.Hour))))
		segmentsAfter(t, dir, 2)

		require.NoError(t, j.Dispatch(t.Context(), v7("c", epoch.Add(25*time.Hour))))
		segmentsAfter(t, dir, 2) // a is over a day older than c.
		require.Equal(t, []string{"b", "c"}, scan(t, j, agent.Cursor{}))
	})

	t.Run("keeps everything since the time instead", func(t *testing.T) {
		dir, j := newJournal(t)
		j.Retain(epoch.Add(time.Hour))

		require.NoError(t, j.Dispatch(t.Context(),
			v7("a", epoch),
			v7("b", epoch.Add(time.Hour)),
			v7("c", epoch.Add(48*time.Hour)),
		))

		// a is before the time, b is the segment containing it. c is two days on, a day would have dropped b.
		segmentsAfter(t, dir, 2)
		require.Equal(t, []string{"b", "c"}, scan(t, j, agent.Cursor{}))
	})

	t.Run("moving the time forward discards what is now before it", func(t *testing.T) {
		dir, j := newJournal(t)
		require.NoError(t, j.Dispatch(t.Context(), v7("a", epoch), v7("b", epoch.Add(time.Hour)), v7("c", epoch.Add(2*time.Hour))))
		segmentsAfter(t, dir, 3)

		j.Retain(epoch.Add(2 * time.Hour))
		segmentsAfter(t, dir, 1)
		require.Equal(t, []string{"c"}, scan(t, j, agent.Cursor{}))
	})

	t.Run("the zero time goes back to the default", func(t *testing.T) {
		dir, j := newJournal(t)
		j.Retain(epoch.Add(48 * time.Hour))
		j.Retain(time.Time{})

		require.NoError(t, j.Dispatch(t.Context(), v7("a", epoch), v7("b", epoch.Add(time.Hour))))
		segmentsAfter(t, dir, 2)
	})
}
