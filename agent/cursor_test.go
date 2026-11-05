package agent_test

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"
)

var epoch = time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC)

func TestCursor(t *testing.T) {
	segment := uuid.Must(uuid.FromString(journal.Segment(epoch)[:36]))

	t.Run("should round trip the segment and offset", func(t *testing.T) {
		c := agent.NewCursor(segment, 123456789)
		require.Equal(t, segment, c.Segment())
		require.Equal(t, int64(123456789), c.Offset())
	})

	t.Run("should be the segment id at offset zero and remain a valid v7", func(t *testing.T) {
		c := agent.NewCursor(segment, 0)
		require.Equal(t, segment, uuid.UUID(c))
		require.Equal(t, uuid.V7, uuid.UUID(c).Version())
		require.Equal(t, uuid.VariantRFC9562, uuid.UUID(c).Variant())
	})

	t.Run("should keep the version and variant when holding a maximum offset", func(t *testing.T) {
		c := agent.NewCursor(segment, 1<<62-1)
		require.Equal(t, int64(1<<62-1), c.Offset())
		require.Equal(t, uuid.V7, uuid.UUID(c).Version())
		require.Equal(t, uuid.VariantRFC9562, uuid.UUID(c).Variant())
		require.Equal(t, segment, c.Segment())
	})

	t.Run("should order by segment then offset", func(t *testing.T) {
		next := uuid.Must(uuid.FromString(journal.Segment(epoch.Add(15 * time.Second))[:36]))
		a, b, c := agent.NewCursor(segment, 5), agent.NewCursor(segment, 900), agent.NewCursor(next, 1)
		require.Equal(t, -1, bytes.Compare(a[:], b[:]))
		require.Equal(t, -1, bytes.Compare(b[:], c[:]))
	})
}

func TestCursorFromMessages(t *testing.T) {
	t.Run("should be the start of the window the message belongs to", func(t *testing.T) {
		m := agent.LogInfo(agent.NewPeer("node"), "message")
		c := agent.CursorOf(m)
		require.Equal(t, agent.WindowCursor(agent.MessageTime(m)), c)
		require.Zero(t, c.Offset())
		require.Zero(t, agent.MessageTime(m).Sub(time.UnixMilli(agent.MessageTime(m).UnixMilli())))
	})

	t.Run("should place v4 messages by their timestamp", func(t *testing.T) {
		m := agent.LogInfo(agent.NewPeer("node"), "message")
		m.Id = uuid.Must(uuid.NewV4()).String()
		m.Ts = epoch.Add(7 * time.Second).Unix()
		require.Equal(t, agent.WindowCursor(epoch), agent.CursorOf(m))
	})

	t.Run("should be zero for messages without a time", func(t *testing.T) {
		require.Equal(t, agent.Cursor{}, agent.CursorOf(&agent.Message{}))
	})

	t.Run("should round trip through bytes and treat anything else as zero", func(t *testing.T) {
		c := agent.NewCursor(agent.WindowCursor(epoch).Segment(), 42)
		require.Equal(t, c, agent.CursorFromBytes(c.Bytes()))
		require.Nil(t, agent.Cursor{}.Bytes())
		require.Equal(t, agent.Cursor{}, agent.CursorFromBytes(nil))
		require.Equal(t, agent.Cursor{}, agent.CursorFromBytes([]byte{1, 2, 3}))
	})
}
