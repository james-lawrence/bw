package quorum

import (
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/gofrs/uuid/v5"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// CommandToMessage ...
func commandToMessage(cmd []byte) (_ *agent.Message, err error) {
	var m agent.Message
	return &m, errors.WithStack(proto.Unmarshal(cmd, &m))
}

// NewWAL ...
func NewWAL(c transcoder) WAL {
	return WAL{
		c: c,
		innerstate: innerstate{
			ctx:     TranscoderContext{State: StateHealthy},
			applied: newRecent(),
		},
		m: &sync.RWMutex{},
	}
}

type innerstate struct {
	ctx       TranscoderContext
	snapshots uint64
	logs      []*agent.Message
	applied   *recent // ids recently applied, retried dispatches are committed more than once.
}

// WAL for the quorum.
type WAL struct {
	innerstate
	c transcoder
	m *sync.RWMutex
}

// Apply log is invoked once a log entry is committed.
// It returns a value which will be made available in the
// ApplyFuture returned by Raft.Apply method if that
// method was called on the same Raft node as the FSM.
func (t *WAL) Apply(l *raft.Log) interface{} {
	if _, err := t.ApplyLog(l); err != nil {
		return err
	}

	return nil
}

// ApplyLog applies the log entry, returning the message it carried, if any.
func (t *WAL) ApplyLog(l *raft.Log) (m *agent.Message, err error) {
	switch l.Type {
	case raft.LogBarrier:
		log.Println("barrier invoked", l.Index, l.Term)
		return nil, nil
	case raft.LogCommand:
		return t.decode(t.ctx, l.Data)
	case raft.LogNoop:
		log.Println("noop invoked", l.Index, l.Term)
		return nil, nil
	default:
		return nil, nil
	}
}

func (t *WAL) decode(ctx TranscoderContext, buf []byte) (m *agent.Message, err error) {
	if m, err = commandToMessage(buf); err != nil {
		return nil, err
	}

	if m.Type == agent.Message_UploadEvent {
		// upload progress is client local, it must never be sent through the quorum.
		err = errors.Errorf("upload progress must not be replicated: %s", m.Id)
		log.Println(err)
		return nil, err
	}

	// a dispatch whose acknowledgement was lost is retried, committing the message again. delivery is
	// at least once, so duplicates are rejected here before the transcoders see them. the sender is told
	// (see agent.IsDuplicate), it is what lets it stop retrying.
	if t.applied.maybe(m.GetId()) {
		return nil, agent.ErrDuplicate(m)
	}

	if err = t.c.Decode(ctx, m); err != nil {
		return nil, err
	}

	t.applied.add(m.GetId())

	if m.Ephemeral {
		return m, nil
	}

	t.m.Lock()
	t.logs = append(t.logs, m)
	t.m.Unlock()

	return m, nil
}

// Snapshot is used to support log compaction. This call should
// return an FSMSnapshot which can be used to save a point-in-time
// snapshot of the FSM. Apply and Snapshot are not called in multiple
// threads, but Apply will be called concurrently with Persist. This means
// the FSM should be implemented in a fashion that allows for concurrent
// updates while a snapshot is happening.
func (t *WAL) Snapshot() (raft.FSMSnapshot, error) {
	return &walSnapshot{wal: t, snapshot: atomic.AddUint64(&t.snapshots, 1), max: len(t.logs)}, nil
}

// Restore is used to restore an FSM from a snapshot. It is not called
// concurrently with any other command. The FSM must discard all previous
// state.
func (t *WAL) Restore(o io.ReadCloser) (err error) {
	var (
		version agent.WALPreamble
		encoded []byte
	)

	defer o.Close()

	log.Output(1, fmt.Sprintln("--------------- restore initiated -------------"))
	defer log.Output(1, fmt.Sprintln("--------------- restore completed -------------"))

	atomic.SwapInt64(&t.innerstate.ctx.State, StateRecovering)
	defer atomic.SwapInt64(&t.innerstate.ctx.State, StateHealthy)

	// tell the transcoders the state they hold is being replaced by the snapshot. the markers are local
	// to this node, they bypass the log and the duplicate filter.
	local := agent.NewPeerZero()
	if err = t.c.Decode(t.ctx, agent.NewRestoreInitiated(local)); err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		if err == nil {
			err = t.c.Decode(t.ctx, agent.NewRestoreCompleted(local))
		}
	}()

	// reset the internal state of the write ahead log.
	t.innerstate = innerstate{
		ctx:     t.ctx,
		logs:    make([]*agent.Message, 0, 128),
		applied: newRecent(),
	}

	// read and discard version message, for future use.
	if err = Decode(o, &version); err != nil {
		log.Println("decode version failure", err)
		return errors.WithStack(err)
	}

	for encoded, err = decodeRaw(o); err == nil; encoded, err = decodeRaw(o) {
		// snapshots written before duplicates were rejected can hold them, they are history not failures.
		if _, err = t.decode(t.ctx, encoded); agent.IsDuplicate(err) {
			continue
		} else if err != nil {
			log.Println("decode event failure", err)
			return errors.WithStack(err)
		}
	}

	return iox.IgnoreEOF(err)
}

func (t *WAL) advance(n int) {
	t.m.Lock()
	t.logs = t.logs[n:]
	t.m.Unlock()
}

type walSnapshot struct {
	snapshot uint64
	wal      *WAL
	min, max int
}

// Persist should dump all necessary state to the WriteCloser 'sink',
// and call sink.Close() when finished or call sink.Cancel() on error.
func (t *walSnapshot) Persist(sink raft.SnapshotSink) (err error) {
	var (
		msg *agent.Message
		i   int
	)

	log.Println("--------------- persist initiated -------------")
	defer log.Println("--------------- persist completed -------------")
	for i, msg = range t.wal.logs[:t.max] {
		switch msg.GetType() {
		// whenever we encounter a deploy command event reset the state,
		// this ensures we only keep the state of the latest deploy when compacting the fsm.
		case agent.Message_DeployCommandEvent:
			t.min = i
		}
	}

	if err = encodeProtoTo(sink, agent.NewWALPreamble()); err != nil {
		sink.Cancel()
		return errors.WithStack(err)
	}

	history := t.wal.logs[t.min:t.max]
	if err = encodeTo(sink, history...); err != nil {
		sink.Cancel()
		return errors.WithStack(err)
	}

	if err = t.wal.c.Encode(sink); err != nil {
		sink.Cancel()
		return err
	}

	return sink.Close()
}

// Release is invoked when we are finished with the snapshot.
func (t walSnapshot) Release() {
	if t.snapshot == t.wal.snapshots {
		t.wal.advance(t.min)
	}
}

const (
	// appliedPeriod how long a generation of the filter is current. two generations are kept, so
	// the filter remembers at least the trailing appliedPeriod.
	appliedPeriod = 12 * time.Hour
	// appliedCapacity the number of messages a generation is sized for. the false positive rate climbs if
	// more than this are applied within a period.
	appliedCapacity      = 20_000
	appliedFalsePositive = 1e-7
)

// recent is a pair of bloom filters over the ids of the messages applied. the current generation is
// replaced every appliedPeriod and becomes the previous, both are checked. a bloom filter has false
// positives, so the rate is kept tiny and senders handle a rejection of a first attempt by assigning
// the message a new id (see agentutil.ReliableDispatch). only touched by the raft FSM goroutine.
type recent struct {
	now               func() time.Time
	rotated           time.Time
	current, previous *bloom.BloomFilter
}

func newRecent() *recent {
	return newRecentAt(time.Now)
}

func newRecentAt(now func() time.Time) *recent {
	return &recent{
		now:      now,
		rotated:  now(),
		current:  bloom.NewWithEstimates(appliedCapacity, appliedFalsePositive),
		previous: bloom.NewWithEstimates(appliedCapacity, appliedFalsePositive),
	}
}

// cycle replaces the generations when the current one is old enough. when more than a whole
// extra period has passed the current generation is stale as well.
func (t *recent) cycle() {
	elapsed := t.now().Sub(t.rotated)
	if elapsed < appliedPeriod {
		return
	}

	if elapsed >= 2*appliedPeriod {
		t.previous = bloom.NewWithEstimates(appliedCapacity, appliedFalsePositive)
	} else {
		t.previous = t.current
	}

	t.current = bloom.NewWithEstimates(appliedCapacity, appliedFalsePositive)
	t.rotated = t.now()
}

// maybe reports whether the id may have been applied.
func (t *recent) maybe(id string) bool {
	if id == "" || id == uuid.Nil.String() {
		return false // can't be told apart.
	}

	t.cycle()

	return t.current.TestString(id) || t.previous.TestString(id)
}

// add records the id.
func (t *recent) add(id string) {
	if id == "" || id == uuid.Nil.String() {
		return
	}

	t.cycle()
	t.current.AddString(id)
}
