package quorum

import (
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
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
			ctx: TranscoderContext{State: StateHealthy},
		},
		m: &sync.RWMutex{},
	}
}

type innerstate struct {
	ctx       TranscoderContext
	snapshots uint64
	logs      []*agent.Message
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
	switch l.Type {
	case raft.LogBarrier:
		log.Println("barrier invoked", l.Index, l.Term)
	case raft.LogCommand:
		if err := t.decode(t.ctx, l.Data); err != nil {
			return err
		}
	case raft.LogNoop:
		log.Println("noop invoked", l.Index, l.Term)
	}

	return nil
}

func (t *WAL) decode(ctx TranscoderContext, buf []byte) error {
	var (
		err error
		m   *agent.Message
	)

	if m, err = commandToMessage(buf); err != nil {
		return err
	}

	if err = t.c.Decode(ctx, m); err != nil {
		return err
	}

	if m.DisallowWAL {
		return nil
	}

	t.m.Lock()
	t.logs = append(t.logs, m)
	t.m.Unlock()

	return nil
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

	log.Output(1, fmt.Sprintln("WAL restoring"))
	defer log.Output(1, fmt.Sprintln("WAL restored"))

	atomic.SwapInt64(&t.innerstate.ctx.State, StateRecovering)
	defer atomic.SwapInt64(&t.innerstate.ctx.State, StateHealthy)

	// reset the internal state of the write ahead log.
	t.innerstate = innerstate{
		ctx:  t.ctx,
		logs: make([]*agent.Message, 0, 128),
	}

	// read and discard version message, for future use.
	if err = Decode(o, &version); err != nil {
		log.Println("decode version failure", err)
		return errors.WithStack(err)
	}

	for encoded, err = decodeRaw(o); err == nil; encoded, err = decodeRaw(o) {
		if err = t.decode(t.ctx, encoded); err != nil {
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

	log.Println("persist invoked")
	defer log.Println("persist completed")
	for i, msg = range t.wal.logs[:t.max] {
		switch msg.GetType() {
		// whenever we encounter a deploy command event reset the state,
		// this ensures we only keep the state of the latest deploy when compacting the fsm.
		case agent.Message_DeployCommandEvent:
			t.min = i
		}
	}

	if err = encodeProtoTo(sink, agentutil.WALPreamble()); err != nil {
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
