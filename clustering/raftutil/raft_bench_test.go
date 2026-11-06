package raftutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/james-lawrence/bw/internal/testingx"
)

// benchFSM is a state machine that remembers the messages it applied and writes them to its snapshots, a stand in
// for the quorum WAL.
type benchFSM struct {
	m         sync.Mutex
	logs      [][]byte
	snapshots atomic.Int64
}

func (t *benchFSM) Apply(l *raft.Log) interface{} {
	t.m.Lock()
	defer t.m.Unlock()
	t.logs = append(t.logs, l.Data)
	return nil
}

func (t *benchFSM) Snapshot() (raft.FSMSnapshot, error) {
	t.m.Lock()
	defer t.m.Unlock()
	t.snapshots.Add(1)
	return benchSnapshot{logs: append([][]byte(nil), t.logs[max(0, len(t.logs)-128):]...)}, nil
}

func (t *benchFSM) Restore(r io.ReadCloser) error { return r.Close() }

type benchSnapshot struct{ logs [][]byte }

func (t benchSnapshot) Persist(sink raft.SnapshotSink) error {
	for _, l := range t.logs {
		if _, err := sink.Write(l); err != nil {
			sink.Cancel()
			return err
		}
	}
	return sink.Close()
}

func (benchSnapshot) Release() {}

// benchMessage roughly the size of a serialized heartbeat or progress message.
var benchMessage = make([]byte, 140)

// benchStore is how the quorum stores its raft state.
type benchStore interface {
	raft.LogStore
	raft.StableStore
}

// benchRaft a single node raft cluster.
func benchRaft(b *testing.B, conf *raft.Config, store benchStore, snapshots raft.SnapshotStore, fsm raft.FSM) *raft.Raft {
	b.Helper()

	conf.LocalID = "bench"
	conf.LogOutput = io.Discard

	_, trans := raft.NewInmemTransport("bench")
	r, err := raft.NewRaft(conf, fsm, store, store, snapshots, trans)
	if err != nil {
		b.Fatal(err)
	}

	if err = r.BootstrapCluster(raft.Configuration{Servers: []raft.Server{{ID: "bench", Address: trans.LocalAddr()}}}).Error(); err != nil {
		b.Fatal(err)
	}

	for deadline := time.Now().Add(10 * time.Second); r.State() != raft.Leader; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			b.Fatal("timed out waiting to become the leader")
		}
	}

	b.Cleanup(func() { r.Shutdown().Error() })

	return r
}

func benchConfig() *raft.Config {
	conf := defaultRaftConfig()
	// single node, don't wait for an election.
	conf.HeartbeatTimeout = 50 * time.Millisecond
	conf.ElectionTimeout = 50 * time.Millisecond
	conf.LeaderLeaseTimeout = 50 * time.Millisecond
	return conf
}

func benchBolt(b *testing.B, nosync bool) *raftboltdb.BoltStore {
	b.Helper()

	s, err := raftboltdb.New(raftboltdb.Options{Path: filepath.Join(testingx.TempDir(b), "state.bin"), NoSync: nosync})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { s.Close() })

	return s
}

// BenchmarkRaftApply every dispatched message is a raft entry, written to the log store before it is committed.
// the quorum uses the bolt store with its defaults (fsync on every write, see commandutils.RaftStoreFilepath).
func BenchmarkRaftApply(b *testing.B) {
	for _, v := range []struct {
		name  string
		store func(*testing.B) benchStore
	}{
		{"current-bolt-fsync", func(b *testing.B) benchStore { return benchBolt(b, false) }},
		{"proposed-bolt-nosync", func(b *testing.B) benchStore { return benchBolt(b, true) }},
		{"proposed-inmem", func(b *testing.B) benchStore { return raft.NewInmemStore() }},
	} {
		b.Run(v.name, func(b *testing.B) {
			r := benchRaft(b, benchConfig(), v.store(b), raft.NewInmemSnapshotStore(), &benchFSM{})

			report := testingx.BenchIO(b)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := r.Apply(benchMessage, 10*time.Second).Error(); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			report(b.N)
		})
	}
}

// BenchmarkPassiveReset the quorum discards its raft state every time it connects (see Protocol.connect and
// cmdopts.Peering.Raft), this is the cost of one reset: removing the directory, opening the stores and starting raft.
func BenchmarkPassiveReset(b *testing.B) {
	dir := filepath.Join(testingx.TempDir(b), "raft.d")

	report := testingx.BenchIO(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := os.RemoveAll(dir); err != nil {
			b.Fatal(err)
		}

		if err := os.MkdirAll(dir, 0700); err != nil {
			b.Fatal(err)
		}

		store, err := raftboltdb.New(raftboltdb.Options{Path: filepath.Join(dir, "state.bin")})
		if err != nil {
			b.Fatal(err)
		}

		snapshots, err := raft.NewFileSnapshotStore(dir, 5, io.Discard)
		if err != nil {
			b.Fatal(err)
		}

		conf := benchConfig()
		conf.LocalID = "bench"
		conf.LogOutput = io.Discard
		_, trans := raft.NewInmemTransport("bench")
		r, err := raft.NewRaft(conf, &benchFSM{}, store, store, snapshots, trans)
		if err != nil {
			b.Fatal(err)
		}

		if err = r.BootstrapCluster(raft.Configuration{Servers: []raft.Server{{ID: "bench", Address: trans.LocalAddr()}}}).Error(); err != nil {
			b.Fatal(err)
		}

		if err = r.Shutdown().Error(); err != nil {
			b.Fatal(err)
		}

		if err = store.Close(); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	report(b.N)
}

// BenchmarkSnapshotThreshold how often the quorum snapshots while messages are applied. raft only evaluates the
// threshold once per SnapshotInterval (30 minutes in the quorum), so the production configuration is compared with
// shorter intervals to isolate the effect of the threshold.
func BenchmarkSnapshotThreshold(b *testing.B) {
	for _, v := range []struct {
		name      string
		interval  time.Duration
		threshold uint64
	}{
		{"current-interval-30m-threshold-256", 30 * time.Minute, 256},
		{"interval-10ms-threshold-256", 10 * time.Millisecond, 256},
		{"interval-10ms-threshold-8192", 10 * time.Millisecond, 8192},
	} {
		b.Run(v.name, func(b *testing.B) {
			conf := benchConfig()
			conf.SnapshotInterval = v.interval
			conf.SnapshotThreshold = v.threshold

			dir := testingx.TempDir(b)
			snapshots, err := raft.NewFileSnapshotStore(dir, 5, io.Discard)
			if err != nil {
				b.Fatal(err)
			}

			fsm := &benchFSM{}
			r := benchRaft(b, conf, raft.NewInmemStore(), snapshots, fsm)

			report := testingx.BenchIO(b)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := r.Apply(benchMessage, 10*time.Second).Error(); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			report(b.N)

			b.ReportMetric(float64(fsm.snapshots.Load()), "snapshots")
			b.Log(fmt.Sprintf("%d messages applied, %d snapshots", b.N, fsm.snapshots.Load()))
		})
	}
}
