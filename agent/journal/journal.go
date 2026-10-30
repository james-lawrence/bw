// Package journal provides an append only, time windowed log of messages.
//
// The raft FSM appends to the journal (a local file write) and observers pull from it
// at their own pace, so a slow observer can never slow the state machine.
package journal

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/asynccompute"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/iterx"
	"github.com/james-lawrence/bw/internal/langx"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

const (
	// Window is the amount of time covered by a single segment.
	Window = 15 * time.Second

	extension = ".log"
)

// segmentID returns the smallest UUIDv7 for the window containing t, the timestamp is
// the start of the window and every other bit is zero (besides version and variant).
func segmentID(t time.Time) uuid.UUID {
	id := uuid.UUID{}
	// 48 bit timestamp in the high bytes, the low 16 bits (version and rand_a) stay zero.
	binary.BigEndian.PutUint64(id[:8], uint64(t.Truncate(Window).UnixMilli())<<16)
	id.SetVersion(uuid.V7)
	id.SetVariant(uuid.VariantRFC9562)

	return id
}

// Segment returns the file name for the window containing t. the name is the smallest UUIDv7
// for the start of the window so that lexicographic order is chronological order.
func Segment(t time.Time) string {
	return segmentName(segmentID(t))
}

func segmentName(id uuid.UUID) string {
	return id.String() + extension
}

func segmentTime(name string) (time.Time, error) {
	id, err := uuid.FromString(strings.TrimSuffix(name, extension))
	if err != nil {
		return time.Time{}, err
	}

	ts, err := uuid.TimestampFromV7(id)
	if err != nil {
		return time.Time{}, err
	}

	return ts.Time()
}

// Option configures a journal.
type Option func(*Journal)

// OptionRetention how long segments are kept, measured from the newest write.
func OptionRetention(d time.Duration) Option {
	return func(j *Journal) {
		j.retention = d
	}
}

type discard struct {
	*Journal
	discard string
}

// New journal stored in the given directory.
func New(dir string, options ...Option) (*Journal, error) {
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, errors.WithStack(err)
	}

	j := new(langx.Clone(Journal{
		dir:       dir,
		retention: time.Hour,
		wake:      make(chan struct{}),
		pruner: asynccompute.New(
			func(_ context.Context, evt discard) error {
				evt.prune(evt.discard)
				return nil
			},
			asynccompute.Workers[discard](1),
			asynccompute.Backlog[discard](4),
		),
	}, options...))

	return j, nil
}

// Close stops pruning and releases the open segment. dispatching to a closed journal is an error.
func (t *Journal) Close() (err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	if t.f != nil {
		err = errors.WithStack(t.f.Close())
		t.f = nil
	}

	return errorsx.Compact(t.pruner.Close(), err)
}

// Journal is safe for concurrent use.
type Journal struct {
	dir       string
	retention time.Duration

	pruner *asynccompute.Pool[discard] // removes expired segments off the apply path.

	m       sync.Mutex
	closed  bool
	current string
	f       *os.File
	wake    chan struct{} // closed and replaced on every write.
}

// Dispatch appends the messages to the journal. every message is recorded, filtering
// (hidden messages) is the responsibility of the readers.
func (t *Journal) Dispatch(_ context.Context, messages ...*agent.Message) (err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if t.closed {
		return errors.New("journal is closed")
	}

	for _, m := range messages {
		if err = t.append(m); err != nil {
			return err
		}
	}

	close(t.wake)
	t.wake = make(chan struct{})

	return nil
}

func (t *Journal) append(m *agent.Message) (err error) {
	var (
		encoded []byte
		f       *os.File
	)

	if encoded, err = proto.Marshal(m); err != nil {
		return errors.WithStack(err)
	}

	record := protowire.AppendVarint(nil, uint64(len(encoded)))
	record = append(record, encoded...)

	ts := agent.MessageTime(m)
	if ts.IsZero() {
		return errors.Errorf("unable to journal message %q: no timestamp, expected a uuidv7 id or ts", m.GetId())
	}

	name := Segment(ts)
	if name == t.current {
		f = t.f
	} else if name > t.current {
		if f, err = t.rollover(name); err != nil {
			return err
		}
	} else {
		// late message for an older window, don't disturb the current segment.
		if f, err = os.OpenFile(filepath.Join(t.dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640); err != nil {
			return errors.WithStack(err)
		}
		defer f.Close()
	}

	// a single write so readers never observe a partially written length prefix and body in separate calls.
	_, err = f.Write(record)
	return errors.WithStack(err)
}

func (t *Journal) rollover(name string) (f *os.File, err error) {
	if f, err = os.OpenFile(filepath.Join(t.dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640); err != nil {
		return nil, errors.WithStack(err)
	}

	if t.f != nil {
		t.f.Close()
	}
	t.f, t.current = f, name

	errorsx.Log(t.pruner.Run(context.Background(), discard{Journal: t, discard: name}))

	return f, nil
}

// prune removes segments older than the retention relative to the newest segment.
func (t *Journal) prune(newest string) {
	now, err := segmentTime(newest)
	if err != nil {
		return
	}

	for _, s := range t.Segments() {
		if ts, err := segmentTime(s); err == nil && now.Sub(ts) > t.retention {
			if err := os.Remove(filepath.Join(t.dir, s)); err != nil {
				log.Println(errors.Wrap(err, "unable to prune journal segment"))
			}
		}
	}
}

// Segments returns the names of the segments in chronological order.
func (t *Journal) Segments() []string {
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), extension) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	return names
}

// Tail returns a cursor pointing at the end of the journal.
func (t *Journal) Tail() agent.Cursor {
	t.m.Lock()
	defer t.m.Unlock()

	segments := t.Segments()
	if len(segments) == 0 {
		return agent.Cursor{}
	}

	last := segments[len(segments)-1]
	info, err := os.Stat(filepath.Join(t.dir, last))
	if err != nil {
		return agent.Cursor{}
	}

	id, err := uuid.FromString(strings.TrimSuffix(last, extension))
	if err != nil {
		return agent.Cursor{}
	}

	return agent.NewCursor(id, info.Size())
}

func (t *Journal) waiter() <-chan struct{} {
	t.m.Lock()
	defer t.m.Unlock()
	return t.wake
}

// Connect streams the messages written after the call.
func (t *Journal) Connect() iterx.Seq[*agent.Message] {
	return t.From(t.Tail())
}

// From streams the messages starting at the cursor. hidden messages are skipped.
// the sequence runs until the context is cancelled or the consumer stops.
func (t *Journal) From(c agent.Cursor) iterx.Seq[*agent.Message] {
	return iterx.New(func(ctx context.Context, yield func(*agent.Message) bool) (err error) {
		for {
			// grab the waiter before reading so writes landing during the read are not missed.
			wake := t.waiter()

			var more bool
			if c, more, err = t.drain(ctx, c, yield); err != nil || !more {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-wake:
			}
		}
	})
}

// drain delivers everything currently in the journal after the cursor. more is false when the
// consumer stopped.
func (t *Journal) drain(ctx context.Context, c agent.Cursor, yield func(*agent.Message) bool) (_ agent.Cursor, more bool, err error) {
	current := segmentName(c.Segment())

	for _, s := range t.Segments() {
		if c != (agent.Cursor{}) && s < current {
			continue
		}

		offset := int64(0)
		if s == current {
			offset = c.Offset()
		}

		var next int64
		next, more, err = t.read(ctx, s, offset, yield)
		if err != nil {
			return c, false, err
		}

		id, err := uuid.FromString(strings.TrimSuffix(s, extension))
		if err != nil {
			return c, false, errors.WithStack(err)
		}

		c = agent.NewCursor(id, next)
		current = s

		if !more {
			return c, false, nil
		}
	}

	return c, true, nil
}

// read records from the segment starting at offset, returns the offset after the last complete record.
// more is false when the consumer stopped.
func (t *Journal) read(ctx context.Context, segment string, offset int64, yield func(*agent.Message) bool) (_ int64, more bool, err error) {
	f, err := os.Open(filepath.Join(t.dir, segment))
	if os.IsNotExist(err) {
		return offset, true, nil // pruned.
	} else if err != nil {
		return offset, false, errors.WithStack(err)
	}
	defer f.Close()

	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return offset, false, errors.WithStack(err)
	}

	r := bufio.NewReader(f)
	for {
		n, err := binary.ReadUvarint(r)
		if err != nil {
			return offset, true, nil // end of data or partial prefix, retry on next wake.
		}

		buf := make([]byte, n)
		if _, err = io.ReadFull(r, buf); err != nil {
			return offset, true, nil // partial record.
		}

		offset += int64(protowire.SizeVarint(n)) + int64(n)

		var m agent.Message
		if err = proto.Unmarshal(buf, &m); err != nil {
			log.Println(errors.Wrap(err, "skipping corrupt journal record"))
			continue
		}

		if m.Hidden {
			continue
		}

		if err = ctx.Err(); err != nil {
			return offset, false, err
		}

		if !yield(&m) {
			return offset, false, nil
		}
	}
}
