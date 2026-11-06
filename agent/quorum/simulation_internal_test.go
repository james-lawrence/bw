package quorum

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/journal"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/clustering/raftutil"
	deployments "github.com/james-lawrence/bw/deployment"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// simulation of the quorum: real raft (in memory), the real deployment decoder and the real
// deploy coordination (deployments.RunDeploy). only the grpc edges are replaced.
//
// clients drop messages that repeat an Id, so a user visible repeat is one logical event that was
// *created* more than once. the invariant checked here is that a logical event, identified by its
// semantic key (type, peer, payload; not the Id or timestamp), is recorded once per deploy.

// recorder is a transcoder capturing every message applied by one node.
type recorder struct {
	m        sync.Mutex
	messages []*agent.Message
}

func (t *recorder) Decode(ctx TranscoderContext, m *agent.Message) error {
	if ctx.State == StateRecovering {
		return nil
	}

	t.m.Lock()
	defer t.m.Unlock()
	t.messages = append(t.messages, proto.Clone(m).(*agent.Message))
	return nil
}

func (t *recorder) Encode(io.Writer) error { return nil }

func (t *recorder) snapshot() []*agent.Message {
	t.m.Lock()
	defer t.m.Unlock()
	return append([]*agent.Message(nil), t.messages...)
}

// key identifies the logical event a message reports, ignoring when/which instance created it.
func key(m *agent.Message) string {
	c := proto.Clone(m).(*agent.Message)
	c.Id, c.Ts = "", 0
	return prototext.MarshalOptions{}.Format(c)
}

// duplicates returns the logical events recorded more than once, with the number of times.
// ignore drops the events that are expected to repeat (heartbeats, periodic status).
func duplicates(messages []*agent.Message, ignore func(*agent.Message) bool) map[string]int {
	counts := map[string]int{}
	for _, m := range messages {
		if ignore != nil && ignore(m) {
			continue
		}
		counts[key(m)]++
	}

	for k, n := range counts {
		if n < 2 {
			delete(counts, k)
		}
	}

	return counts
}

func repeatable(m *agent.Message) bool {
	return m.Type == agent.Message_DeployHeartbeat
}

type simcluster struct {
	nodes     []*simnode
	transport []*raft.InmemTransport
}

type simnode struct {
	name       string
	raft       *raft.Raft
	deployment *deployment
	rec        *recorder
}

func newSimCluster(t *testing.T, names ...string) *simcluster {
	t.Helper()

	var (
		servers []raft.Server
		c       = &simcluster{}
	)

	for _, name := range names {
		cfg := raft.DefaultConfig()
		cfg.LogOutput = io.Discard
		cfg.LocalID = raft.ServerID(name)
		store := raft.NewInmemStore()
		addr, transport := raft.NewInmemTransport("")

		n := &simnode{
			name:       name,
			deployment: newDeployment(newSimPeers(0)),
			rec:        &recorder{},
		}

		jrnl, err := journal.New(t.TempDir())
		require.NoError(t, err)
		t.Cleanup(func() { jrnl.Close() })

		wal := NewWAL(NewTranscoder(n.deployment, n.rec, NewObserver(jrnl)))
		r, err := raft.NewRaft(cfg, &wal, store, store, raft.NewInmemSnapshotStore(), transport)
		require.NoError(t, err)
		t.Cleanup(func() { r.Shutdown() })

		n.raft = r
		c.nodes = append(c.nodes, n)
		c.transport = append(c.transport, transport)
		servers = append(servers, raft.Server{Address: addr, ID: cfg.LocalID, Suffrage: raft.Voter})
	}

	for i, n := range c.nodes {
		for _, o := range c.transport {
			c.transport[i].Connect(o.LocalAddr(), o)
		}
		require.NoError(t, n.raft.BootstrapCluster(raft.Configuration{Servers: servers}).Error())
	}

	return c
}

func (t *simcluster) leader(tt *testing.T) *simnode {
	tt.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		for _, n := range t.nodes {
			if n.raft.State() == raft.Leader {
				return n
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	tt.Fatal("no leader elected")
	return nil
}

// await blocks until every node has applied the index of the leader.
func (t *simcluster) settle(tt *testing.T) {
	tt.Helper()

	l := t.leader(tt)
	require.NoError(tt, l.raft.Barrier(5*time.Second).Error())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ok := true
		for _, n := range t.nodes {
			if n.raft.AppliedIndex() < l.raft.LastIndex() {
				ok = false
			}
		}
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// simpeers is a static cluster of the given size.
type simpeers struct {
	local *agent.Peer
	peers []*agent.Peer
}

func newSimPeers(n int) simpeers {
	s := simpeers{local: agent.NewPeer("leader")}
	for i := 0; i < n; i++ {
		s.peers = append(s.peers, agent.NewPeer(fmt.Sprintf("node-%d", i)))
	}
	return s
}

func (t simpeers) Members() []*memberlist.Node         { return nil }
func (t simpeers) Get([]byte) *memberlist.Node         { return nil }
func (t simpeers) GetN(int, []byte) []*memberlist.Node { return nil }
func (t simpeers) Local() *agent.Peer                  { return t.local }
func (t simpeers) Quorum() []*agent.Peer               { return nil }
func (t simpeers) Peers() []*agent.Peer                { return t.peers }

// simdeploy runs the leader side coordination, the same call StateMachine.Deploy backgrounds.
// visits counts the deploys dispatched to each node.
func simdeploy(ctx context.Context, sm *StateMachine, c simpeers, visits *sync.Map) (int64, bool) {
	return deployments.RunDeploy(c.Local(), c, sm,
		deployments.DeployOptionChecker(deployments.OperationFunc(func(context.Context, *agent.Peer) (*agent.Deploy, error) {
			return &agent.Deploy{Stage: agent.Deploy_Completed}, nil
		})),
		deployments.DeployOptionDeployer(deployments.OperationFunc(func(_ context.Context, p *agent.Peer) (*agent.Deploy, error) {
			n, _ := visits.LoadOrStore(p.Name, new(int64))
			atomic.AddInt64(n.(*int64), 1)
			return &agent.Deploy{Stage: agent.Deploy_Deploying}, nil
		})),
		deployments.DeployOptionHeartbeatFrequency(time.Hour),
		deployments.DeployOptionMonitor(deployments.NewMonitor(deployments.MonitorTicklerPeriodic(20*time.Millisecond))),
	)
}

func begin(t *testing.T, sm *StateMachine, c simpeers) {
	t.Helper()
	cmd := agent.DeployCommandBegin("tester", &agent.Archive{DeploymentID: []byte("d1")}, &agent.DeployOptions{})
	require.NoError(t, sm.Dispatch(context.Background(), agent.NewDeployCommand(c.Local(), cmd)))
}

func TestSimulation(t *testing.T) {
	t.Run("a single deploy records each event exactly once", func(t *testing.T) {
		cl := newSimCluster(t, "a", "b", "c")
		l := cl.leader(t)
		sm := NewMachine(agent.NewPeer("leader"), l.raft)
		c := newSimPeers(3)

		begin(t, sm, c)
		_, ok := simdeploy(context.Background(), sm, c, &sync.Map{})
		require.True(t, ok)
		cl.settle(t)

		for _, n := range cl.nodes {
			require.Empty(t, duplicates(n.rec.snapshot(), repeatable), "node %s", n.name)
		}
	})

	// two coordinators for the same deploy, which is what an orphaned RunDeploy (old leader, nothing
	// cancels it when leadership moves) plus the new leader's restartActiveDeploy produce.
	// the fsm cannot tell them apart: every event is created twice with its own Id, so clients
	// can't drop either. this proves the FSM does not protect against the second coordinator,
	// the guard has to be in the producer.
	t.Run("two coordinators for one deploy repeat every event", func(t *testing.T) {
		t.Skip("pending: StateMachine.Deploy has no leadership bound on its RunDeploy goroutine, nothing proves a second one is started yet")

		cl := newSimCluster(t, "a", "b", "c")
		l := cl.leader(t)
		sm := NewMachine(agent.NewPeer("leader"), l.raft)
		c := newSimPeers(3)

		begin(t, sm, c)

		visits := &sync.Map{}
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				simdeploy(context.Background(), sm, c, visits)
			}()
		}
		wg.Wait()
		cl.settle(t)

		visits.Range(func(k, v any) bool {
			require.EqualValues(t, 1, atomic.LoadInt64(v.(*int64)), "node %s was deployed to more than once", k)
			return true
		})
		require.Empty(t, duplicates(l.rec.snapshot(), repeatable))
	})

	// leader observations are handled after the fact, so a node that is already leading can see
	// several. each one used to run restartActiveDeploy, repeating the recovery events with fresh Ids.
	t.Run("repeated leader observations do not repeat the recovery events", func(t *testing.T) {
		jrnl, err := journal.New(t.TempDir())
		require.NoError(t, err)
		t.Cleanup(func() { jrnl.Close() })

		rec := &recorder{}
		c := newSimPeers(0)
		q := New(jrnl, c, rec, nil, raftutil.Protocol{})

		cfg := raft.DefaultConfig()
		cfg.LogOutput = io.Discard
		cfg.LocalID = "a"
		store := raft.NewInmemStore()
		addr, transport := raft.NewInmemTransport("")
		r, err := raft.NewRaft(cfg, q.wal, store, store, raft.NewInmemSnapshotStore(), transport)
		require.NoError(t, err)
		t.Cleanup(func() { r.Shutdown() })
		require.NoError(t, r.BootstrapCluster(raft.Configuration{Servers: []raft.Server{{Address: addr, ID: cfg.LocalID, Suffrage: raft.Voter}}}).Error())
		for r.State() != raft.Leader {
			time.Sleep(10 * time.Millisecond)
		}

		// an active deploy, as recorded by the fsm.
		sm := NewMachine(c.Local(), r)
		begin(t, sm, c)
		require.NotNil(t, q.deployment.getRunningDeploy())

		for i := 0; i < 3; i++ {
			q.leading(r)
		}

		// the recovery fails (no network) but the events it emits are what matter.
		require.Eventually(t, func() bool {
			for _, m := range rec.snapshot() {
				if m.GetDeployCommand().GetCommand() == agent.DeployCommand_Cancel {
					return true
				}
			}
			return false
		}, 10*time.Second, 10*time.Millisecond)
		time.Sleep(500 * time.Millisecond)

		require.Empty(t, duplicates(rec.snapshot(), repeatable))
	})

	// ReliableDispatch retries until it sees success. a write that committed but whose acknowledgement was
	// lost (timeout, leadership change, dropped connection) is retried, and committed again with the same Id.
	// clients only dedupe inside a replay window, so a live duplicate is shown twice.
	t.Run("a dispatch retried after it committed is recorded once", func(t *testing.T) {
		cl := newSimCluster(t, "a", "b", "c")
		l := cl.leader(t)
		sm := NewMachine(agent.NewPeer("leader"), l.raft)
		peer := agent.NewPeer("leader")

		lossy := &ackLoss{inner: sm, losses: 1}
		batch := []*agent.Message{agent.LogInfo(peer, "first"), agent.LogInfo(peer, "second")}

		require.NoError(t, agentutil.ReliableDispatch(context.Background(), lossy, batch...))
		require.NoError(t, sm.Dispatch(context.Background(), agent.LogInfo(peer, "other")))
		cl.settle(t)

		for _, n := range cl.nodes {
			ids := map[string]int{}
			for _, m := range n.rec.snapshot() {
				ids[m.Id]++
			}
			for id, count := range ids {
				require.Equal(t, 1, count, "node %s recorded message %s %d times", n.name, id, count)
			}
		}
	})
}

// ackLoss commits the messages and then reports a failure, as a lost acknowledgement does.
type ackLoss struct {
	inner  *StateMachine
	losses int
}

func (t *ackLoss) Dispatch(ctx context.Context, m ...*agent.Message) error {
	if err := t.inner.Dispatch(ctx, m...); err != nil {
		return err
	}

	if t.losses > 0 {
		t.losses--
		return status.Error(codes.Unavailable, "acknowledgement lost")
	}

	return nil
}

func TestWALDuplicates(t *testing.T) {
	apply := func(w *WAL, m *agent.Message) error {
		data, err := proto.Marshal(m)
		require.NoError(t, err)
		_, err = w.ApplyLog(&raft.Log{Type: raft.LogCommand, Data: data})
		return err
	}

	t.Run("rejects a duplicate and tells the sender", func(t *testing.T) {
		rec := &recorder{}
		w := NewWAL(NewTranscoder(rec))

		m := agent.LogInfo(agent.NewPeer("p"), "once")
		require.NoError(t, apply(&w, m))
		err := apply(&w, m)
		require.True(t, agent.IsDuplicate(err), "%v", err)
		require.Len(t, rec.snapshot(), 1)
	})

	t.Run("a message the transcoders rejected can be retried", func(t *testing.T) {
		w := NewWAL(NewTranscoder(Discard{Cause: io.ErrUnexpectedEOF}))

		m := agent.LogInfo(agent.NewPeer("p"), "once")
		require.ErrorIs(t, apply(&w, m), io.ErrUnexpectedEOF)
		require.ErrorIs(t, apply(&w, m), io.ErrUnexpectedEOF)
	})

	t.Run("messages without an id are never duplicates", func(t *testing.T) {
		rec := &recorder{}
		w := NewWAL(NewTranscoder(rec))

		m := agent.LogInfo(agent.NewPeer("p"), "once")
		m.Id = ""
		require.NoError(t, apply(&w, m))
		require.NoError(t, apply(&w, m))
	})

	t.Run("remembers the trailing period across cycles", func(t *testing.T) {
		clock := time.Now()
		r := newRecentAt(func() time.Time { return clock })

		r.add("first")
		clock = clock.Add(appliedPeriod + time.Minute)
		r.add("second")
		require.True(t, r.maybe("first"), "previous generation")
		require.True(t, r.maybe("second"))

		clock = clock.Add(appliedPeriod)
		require.False(t, r.maybe("first"), "older than the trailing period")
		require.True(t, r.maybe("second"), "now the previous generation")

		clock = clock.Add(5 * appliedPeriod)
		require.False(t, r.maybe("second"), "idle for longer than both generations")
	})

	t.Run("false positives are rare enough to be reported", func(t *testing.T) {
		r := newRecent()
		for i := 0; i < appliedCapacity; i++ {
			r.add(fmt.Sprintf("seen-%d", i))
		}
		hits := 0
		for i := 0; i < 100000; i++ {
			if r.maybe(fmt.Sprintf("new-%d", i)) {
				hits++
			}
		}
		require.Zero(t, hits)
	})
}

func TestReliableDispatchDuplicate(t *testing.T) {
	// the filter wrongly rejects the second message of a batch on its first attempt. it must get a new id
	// and be delivered, the others must keep theirs and be delivered once.
	t.Run("a first attempt rejected as a duplicate is delivered under a new id", func(t *testing.T) {
		peer := agent.NewPeer("p")
		batch := []*agent.Message{agent.LogInfo(peer, "a"), agent.LogInfo(peer, "b"), agent.LogInfo(peer, "c")}
		ids := []string{batch[0].Id, batch[1].Id, batch[2].Id}
		d := &filterDispatcher{falsepositive: map[string]bool{batch[1].Id: true}, seen: map[string]bool{}}

		require.NoError(t, agentutil.ReliableDispatch(context.Background(), d, batch...))

		require.Equal(t, []string{"a", "b", "c"}, d.delivered)
		require.Equal(t, ids[0], batch[0].Id)
		require.NotEqual(t, ids[1], batch[1].Id)
		require.Equal(t, ids[2], batch[2].Id)
	})
}

// filterDispatcher behaves like the leader: applies what is new, rejects what was seen, and the
// ids in falsepositive are rejected the first time they are offered even though they are new.
type filterDispatcher struct {
	falsepositive map[string]bool
	seen          map[string]bool
	delivered     []string
}

func (t *filterDispatcher) Dispatch(_ context.Context, ms ...*agent.Message) error {
	for _, m := range ms {
		if t.falsepositive[m.Id] || t.seen[m.Id] {
			delete(t.falsepositive, m.Id)
			return agent.ErrDuplicate(m)
		}
		t.seen[m.Id] = true
		t.delivered = append(t.delivered, m.GetLog().GetLog())
	}

	return nil
}
