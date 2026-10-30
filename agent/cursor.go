package agent

import (
	"encoding/binary"

	"github.com/gofrs/uuid/v5"
)

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
