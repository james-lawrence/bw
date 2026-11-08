// Package journal provides an append only, time windowed log of messages.
//
// The raft FSM appends to the journal (a local file write) and observers pull from it
// at their own pace, so a slow observer can never slow the state machine.
package journal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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
	Window = agent.CursorWindow

	// DefaultRetention how long segments are kept when no deploy has retained them, see Retain.
	DefaultRetention = 24 * time.Hour

	extension = ".log"

	// ErrReset ends the followers of a journal that was reset, the messages they were following are gone.
	// they resume from a window cursor, which is still valid.
	ErrReset = errorsx.String("journal reset")
)

// Segment returns the file name for the window containing t. the name is the smallest UUIDv7
// for the start of the window so that lexicographic order is chronological order.
func Segment(t time.Time) string {
	return segmentName(agent.WindowCursor(t).Segment())
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

// OptionRetention how long segments are kept when nothing has been retained, measured from the newest write.
func OptionRetention(d time.Duration) Option {
	return func(j *Journal) {
		j.retention = d
	}
}

// OptionDebounce how long followers coalesce writes for, a follower woken by a write waits this long before reading
// so a burst of writes is delivered by a single pass.
func OptionDebounce(d time.Duration) Option {
	return func(j *Journal) {
		j.debounce = d
	}
}

type discard struct {
	*Journal
	discard string
}

// New journal stored in the given directory, anything already in the directory is removed.
func New(dir string, options ...Option) (*Journal, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, errors.WithStack(err)
	}

	j := new(langx.Clone(Journal{
		dir:       dir,
		retention: DefaultRetention,
		debounce:  200 * time.Millisecond,
		pruner: asynccompute.New(
			func(_ context.Context, evt discard) error {
				evt.prune(evt.discard)
				return nil
			},
			asynccompute.Workers[discard](1),
			asynccompute.Backlog[discard](4),
		),
	}, options...))

	// the cond is built after the clone, it has to reference the mutex of the journal returned.
	j.written = sync.NewCond(&j.m)

	return j, nil
}

// Retain keeps the messages written since the given time, segments entirely before it are discarded. a
// deploy retains from its archive, so the history of the last successful deploy survives for as long as it is the
// latest. the zero time is -inf: back to the retention measured from the newest write.
func (t *Journal) Retain(since time.Time) {
	if since.IsZero() {
		t.floor.Store(0)
	} else {
		t.floor.Store(since.UnixNano())
	}

	t.m.Lock()
	current := t.current
	closed := t.closed
	t.m.Unlock()

	if closed || current == "" {
		return
	}

	errorsx.Log(t.pruner.Run(context.Background(), discard{Journal: t, discard: current}))
}

// Reset discards every message and ends the followers with ErrReset. the state the journal recorded is being
// replaced, e.g. by restoring a snapshot, and the messages that follow rebuild it. replayed messages written into
// the segments followers are reading would be delivered to them again as new events.
func (t *Journal) Reset() (err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if t.closed {
		return errors.New("journal is closed")
	}

	if t.f != nil {
		errorsx.Log(t.f.Close())
		t.f = nil
	}

	for _, s := range t.Segments() {
		if err = os.Remove(filepath.Join(t.dir, s)); err != nil && !os.IsNotExist(err) {
			return errors.WithStack(err)
		}
	}

	t.current, t.tail = "", agent.Cursor{}
	t.epoch++
	t.written.Broadcast() // release the followers, they are over.

	return nil
}

// Close stops pruning and releases the open segment. dispatching to a closed journal is an error.
func (t *Journal) Close() (err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true
	t.written.Broadcast() // release the followers waiting for more data.

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
	debounce  time.Duration
	floor     atomic.Int64 // unix nanoseconds, segments entirely before it are discarded. zero falls back to the retention.
	epoch     uint64       // incremented whenever the journal is reset, followers of an older epoch are over.

	pruner *asynccompute.Pool[discard] // removes expired segments off the apply path.

	m       sync.Mutex
	closed  bool
	current string
	f       *os.File
	tail    agent.Cursor // one past the last message written to the newest segment, where the next record starts.
	written *sync.Cond   // broadcast when tail moves or the journal closes, over m.
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

	t.written.Broadcast()

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
	// a late message for an older window (clock skew, commit latency) goes to the current segment. followers only
	// read forward from their cursor so a record in an older segment would never be delivered.
	if name > t.current {
		if _, err = t.rollover(name); err != nil {
			return err
		}
	}
	f, name = t.f, t.current

	// a single write so readers never observe a partially written length prefix and body in separate calls.
	if _, err = f.Write(record); err != nil {
		return errors.WithStack(err)
	}

	segment, err := segmentTime(name)
	if err != nil {
		return err
	}

	seg := agent.WindowCursor(segment).Segment()

	var offset int64
	if t.tail.Segment() == seg {
		offset = t.tail.Offset()
	}

	t.tail = agent.NewCursor(seg, offset+int64(len(record)))

	return nil
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

// prune removes the segments that are no longer retained: entirely before the floor, or when there isn't one
// older than the retention relative to the newest segment.
func (t *Journal) prune(newest string) {
	now, err := segmentTime(newest)
	if err != nil {
		return
	}

	// the pruner runs while the journal is locked by the writer, it must not take the lock.
	floor := t.floor.Load()

	expired := func(ts time.Time) bool {
		if floor != 0 {
			return ts.Add(Window).UnixNano() <= floor
		}

		return now.Sub(ts) > t.retention
	}

	for _, s := range t.Segments() {
		if ts, err := segmentTime(s); err == nil && expired(ts) {
			if err := os.Remove(filepath.Join(t.dir, s)); err != nil && !os.IsNotExist(err) {
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

// Tail returns a cursor pointing at the end of the journal, one past the last message written.
func (t *Journal) Tail() agent.Cursor {
	t.m.Lock()
	defer t.m.Unlock()

	return t.tail
}

// checkEpoch returns ErrReset when the journal was reset since the epoch.
func (t *Journal) checkEpoch(epoch uint64) error {
	t.m.Lock()
	defer t.m.Unlock()

	if t.epoch != epoch {
		return ErrReset
	}

	return nil
}

// await blocks until the journal has data past the cursor, it is reset, closed, or the context is done.
func (t *Journal) await(ctx context.Context, epoch uint64, c agent.Cursor) error {
	t.m.Lock()
	defer t.m.Unlock()

	for bytes.Compare(t.tail[:], c[:]) <= 0 {
		if t.epoch != epoch {
			return ErrReset
		}

		if t.closed {
			return errors.New("journal is closed")
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		t.written.Wait()
	}

	return nil
}

// Connect streams the messages starting at the cursor. a cursor into a window continues from the cursor, windows
// pruned in the meantime are skipped. hidden messages are skipped.
//
// when tailing the zero cursor streams the messages written after the call and the sequence runs until the
// context is cancelled, the consumer stops, or the journal is reset (ErrReset). otherwise the zero cursor starts at
// the oldest message and the sequence ends when it reaches the end of the journal, it doesn't wait for more.
func (t *Journal) Connect(c agent.Cursor, tail bool) iterx.Seq[*agent.Message] {
	if !tail {
		return iterx.New(func(ctx context.Context, yield func(*agent.Message) bool) error {
			_, _, err := t.drain(ctx, c, yield)
			return err
		})
	}

	t.m.Lock()
	epoch := t.epoch
	if c == (agent.Cursor{}) {
		c = t.tail
	}
	t.m.Unlock()

	return iterx.New(func(ctx context.Context, yield func(*agent.Message) bool) error {
		return t.follow(ctx, epoch, c, yield)
	})
}

func (t *Journal) follow(ctx context.Context, epoch uint64, c agent.Cursor, yield func(*agent.Message) bool) (err error) {
	// Cond.Wait can't observe the context, ring the doorbell when it is done so the waiters re-check it. the lock is
	// required: await checks the context and registers for the broadcast under it, without it a cancellation landing
	// in between would be missed.
	stop := context.AfterFunc(ctx, func() {
		t.m.Lock()
		defer t.m.Unlock()
		t.written.Broadcast()
	})
	defer stop()

	for {
		if err = t.checkEpoch(epoch); err != nil {
			return err
		}

		var more bool
		if c, more, err = t.drain(ctx, c, yield); err != nil || !more {
			return err
		}

		if err = t.await(ctx, epoch, c); err != nil {
			return err
		}

		// coalesce the writes of the window into a single pass. the window starts with the first wake and isn't
		// extended by further writes, otherwise a steady stream of writes would starve the follower.
		timer := time.NewTimer(t.debounce)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
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
