package agent

import (
	"encoding/binary"
	"time"

	"github.com/gofrs/uuid/v5"
)

// CursorWindow is the amount of time covered by a single journal segment.
const CursorWindow = 15 * time.Second

// offsetMask the 62 bits of rand_b, which hold the byte offset within the segment.
const offsetMask = uint64(1)<<62 - 1

// Cursor is a position within the journal. it is a UUIDv7 whose timestamp is the start of the
// segment's window and whose rand_b bits (62) hold the byte offset into the segment. at offset zero
// it is the segment's file name. cursors are ordered by bytes.Compare. the zero value is the
// beginning of the journal.
type Cursor [16]byte

// NewCursor for the segment (see Segment) and the byte offset within it.
func NewCursor(segment uuid.UUID, offset int64) Cursor {
	c := Cursor(segment)
	binary.BigEndian.PutUint64(c[8:], binary.BigEndian.Uint64(c[8:])&^offsetMask|uint64(offset)&offsetMask)
	return c
}

// Segment of the cursor with the offset cleared.
func (t Cursor) Segment() uuid.UUID {
	binary.BigEndian.PutUint64(t[8:], binary.BigEndian.Uint64(t[8:])&^offsetMask)
	return uuid.UUID(t)
}

// Offset in bytes within the segment.
func (t Cursor) Offset() int64 {
	return int64(binary.BigEndian.Uint64(t[8:]) & offsetMask)
}

// WindowCursor is the cursor at the start of the window containing t. the timestamp is the start
// of the window and every other bit is zero (besides version and variant), the smallest UUIDv7 for the window.
func WindowCursor(t time.Time) Cursor {
	id := uuid.UUID{}
	binary.BigEndian.PutUint64(id[:8], uint64(t.Truncate(CursorWindow).UnixMilli())<<16)
	id.SetVersion(uuid.V7)
	id.SetVariant(uuid.VariantRFC9562)

	return Cursor(id)
}

// CursorOf the start of the window the message belongs to, see MessageTime. the zero cursor
// when the message has no time.
func CursorOf(m *Message) Cursor {
	ts := MessageTime(m)
	if ts.IsZero() {
		return Cursor{}
	}

	return WindowCursor(ts)
}

// CursorFromBytes decodes a cursor from its wire form, the zero cursor if it is not one.
func CursorFromBytes(b []byte) (c Cursor) {
	if len(b) == len(c) {
		copy(c[:], b)
	}

	return c
}

// Bytes the wire form of the cursor, nil for the zero cursor.
func (t Cursor) Bytes() []byte {
	if t == (Cursor{}) {
		return nil
	}

	return t[:]
}
