package quorum_test

import (
	"context"
	"errors"
	"io"
	"log"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"google.golang.org/protobuf/proto"

	"github.com/james-lawrence/bw/agent"

	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/journal"
	. "github.com/james-lawrence/bw/agent/quorum"

	"github.com/stretchr/testify/require"
)

type mockLocal struct{}

func (mockLocal) Local() *agent.Peer {
	return agent.NewPeer("")
}

type mockdeploy struct {
	lastOptions *agent.DeployOptions
	lastArchive *agent.Archive
	lastPeers   []agent.Peer
}

func (t *mockdeploy) Deploy(_ dialers.T, opts *agent.DeployOptions, archive *agent.Archive, peers ...agent.Peer) error {
	t.lastOptions = opts
	t.lastArchive = archive
	t.lastPeers = peers
	return nil
}

func newCluster(c Transcoder, names ...string) (peers []*raft.Raft, transports []*raft.InmemTransport, err error) {
	var (
		servers []raft.Server
	)

	for _, n := range names {
		var (
			server    raft.Server
			transport *raft.InmemTransport
			protocol  *raft.Raft
		)

		if server, transport, protocol, err = newPeer(c, n, false); err != nil {
			return peers, transports, err
		}

		peers = append(peers, protocol)
		servers = append(servers, server)
		transports = append(transports, transport)
	}

	for i := range peers {
		p, t := peers[i], transports[i]
		connect(t, transports...)
		config := raft.Configuration{Servers: servers}
		if err = p.BootstrapCluster(config).Error(); err != nil {
			return peers, transports, err
		}
	}

	return peers, transports, nil
}

func newPeer(c Transcoder, name string, leader bool) (raft.Server, *raft.InmemTransport, *raft.Raft, error) {
	var (
		addr      raft.ServerAddress
		transport *raft.InmemTransport
	)
	config := raft.DefaultConfig()
	config.LogOutput = io.Discard
	config.LocalID = raft.ServerID(name)
	storage := raft.NewInmemStore()
	snapshot := raft.NewInmemSnapshotStore()
	addr, transport = raft.NewInmemTransport("")
	fsm := NewWAL(c)
	protocol, err := raft.NewRaft(config, &fsm, storage, storage, snapshot, transport)
	return raft.Server{Address: addr, ID: config.LocalID, Suffrage: raft.Voter}, transport, protocol, err
}

func connect(local *raft.InmemTransport, peers ...*raft.InmemTransport) {
	for _, p := range peers {
		local.Connect(p.LocalAddr(), p)
		p.Connect(local.LocalAddr(), local)
	}
}

func findFirstState(s raft.RaftState, protocols ...*raft.Raft) *raft.Raft {
	for {
		for _, p := range protocols {
			if p.State() == s {
				return p
			}
		}
		runtime.Gosched()
	}
}

func qCommand(d agent.DeployCommand_Command) *agent.DeployCommand {
	return &agent.DeployCommand{
		Command: d,
		Archive: &agent.Archive{},
		Options: &agent.DeployOptions{},
	}
}

func TestStateMachine(t *testing.T) {
	local := mockLocal{}

	t.Run("should write to WAL on dispatch", func(t *testing.T) {
		protocols, _, err := newCluster(NewTranscoder(), "server1", "server2", "server3")
		require.NoError(t, err)

		log.Println("awaiting leader")
		leader := findFirstState(raft.Leader, protocols...)
		log.Println("leader elected")

		lp := agent.NewPeer("node")
		sm := NewMachine(
			lp,
			leader,
		)

		cmd := qCommand(agent.DeployCommand_Begin)
		require.NoError(t, sm.Dispatch(context.Background(), agent.NewDeployCommand(lp, cmd)))
	})

	t.Run("persisting state", func(t *testing.T) {
		for _, c := range []struct {
			name     string
			n        int
			messages []*agent.Message
		}{
			{
				"example 1",
				4,
				[]*agent.Message{
					agent.LogInfo(local.Local(), "message 1"),
					agent.LogInfo(local.Local(), "message 2"),
					agent.LogInfo(local.Local(), "message 3"),
					agent.LogInfo(local.Local(), "message 4"),
					agent.NewDeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
					agent.LogInfo(local.Local(), "message 5"),
					agent.LogInfo(local.Local(), "message 6"),
				},
			},
			{
				"example 2",
				6,
				[]*agent.Message{
					agent.LogInfo(local.Local(), "message 1"),
					agent.LogInfo(local.Local(), "message 2"),
					agent.NewDeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
					agent.LogInfo(local.Local(), "message 3"),
					agent.NewDeployCommand(local.Local(), qCommand(agent.DeployCommand_Done)),
					agent.LogInfo(local.Local(), "message 4"),
					agent.NewDeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
					agent.LogInfo(local.Local(), "message 5"),
					agent.LogInfo(local.Local(), "message 6"),
				},
			},
			{
				"example 3",
				0,
				[]*agent.Message{
					agent.LogInfo(local.Local(), "message 1"),
					agent.LogInfo(local.Local(), "message 2"),
				},
			},
			{
				"example 4",
				0,
				[]*agent.Message{
					agent.NewDeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
				},
			},
			{
				"example 5",
				7,
				[]*agent.Message{
					agent.LogInfo(local.Local(), "message 1"),
					agent.LogInfo(local.Local(), "message 2"),
					agent.LogInfo(local.Local(), "message 3"),
					agent.LogInfo(local.Local(), "message 4"),
					agent.NewDeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
					agent.LogInfo(local.Local(), "message 5"),
					agent.LogInfo(local.Local(), "message 6"),
					agent.NewDeployCommand(local.Local(), qCommand(agent.DeployCommand_Done)),
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				var (
					decoded []*agent.Message
				)

				protocols, _, err := newCluster(NewTranscoder(), "server1", "server2", "server3")
				require.NoError(t, err)

				leader := findFirstState(raft.Leader, protocols...)
				sm := NewMachine(
					agent.NewPeer("node"),
					leader,
				)

				require.NoError(t, sm.Dispatch(context.Background(), c.messages...))

				snapshotfuture := leader.Snapshot()
				require.NoError(t, snapshotfuture.Error())
				_, ior, err := snapshotfuture.Open()
				require.NoError(t, err)

				preamble := &agent.WALPreamble{}
				require.NoError(t, Decode(ior, preamble))
				require.Equal(t, int32(1), preamble.Major)
				require.Equal(t, int32(0), preamble.Minor)
				require.Equal(t, int32(0), preamble.Patch)

				decoded, err = DecodeEvery(ior)
				require.NoError(t, err)

				// ephemeral messages are never written to the WAL, so they are not part of snapshots.
				expected := slices.DeleteFunc(slices.Clone(c.messages[c.n:]), func(m *agent.Message) bool { return m.Ephemeral })
				require.True(t,
					proto.Equal(
						agent.NewLogHistoryEvent(decoded...),
						agent.NewLogHistoryEvent(expected...),
					),
				)
			})
		}
	})

	t.Run("should return an error when dispatch fails", func(t *testing.T) {
		cmd := qCommand(agent.DeployCommand_Begin)
		protocols, _, err := newCluster(NewTranscoder(Discard{Cause: errors.New("boom")}), "server1", "server2", "server3")
		require.NoError(t, err)

		leader := findFirstState(raft.Leader, protocols...)
		sm := NewMachine(
			agent.NewPeer("node"),
			leader,
		)

		require.Error(t, sm.Dispatch(context.Background(), agent.NewDeployCommand(local.Local(), cmd)))
	})

	t.Run("should write message to the observer", func(t *testing.T) {
		messages := []*agent.Message{
			agent.LogInfo(local.Local(), "message 1"),
			agent.LogInfo(local.Local(), "message 2"),
			agent.LogInfo(local.Local(), "message 3"),
			agent.LogInfo(local.Local(), "message 4"),
		}

		obs := make(chan *agent.Message, len(messages))
		jrnl, err := journal.New(t.TempDir())
		require.NoError(t, err)
		t.Cleanup(func() { jrnl.Close() })

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			for m := range jrnl.Connect(agent.Cursor{}).Each(ctx) {
				obs <- m
			}
		}()

		ob := NewObserver(jrnl)
		protocols, _, err := newCluster(NewTranscoder(ob), "server1")
		require.NoError(t, err)

		leader := findFirstState(raft.Leader, protocols...)
		sm := NewMachine(
			agent.NewPeer("node"),
			leader,
		)

		require.NoError(t, sm.Dispatch(context.Background(), messages...))

		for _, m := range messages {
			expected := <-obs
			require.True(t, proto.Equal(expected, m))
		}
	})

	// regression: observers are invoked from the raft FSM, a stalled observer must not throttle the WAL
	// otherwise the apply queue fills and writes fail with "timed out enqueuing operation".
	t.Run("should not be throttled by stalled observers", func(t *testing.T) {
		jrnl, err := journal.New(t.TempDir())
		require.NoError(t, err)
		t.Cleanup(func() { jrnl.Close() })

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stalled := make(chan *agent.Message) // never read.
		go func() {
			for m := range jrnl.Connect(agent.Cursor{}).Each(ctx) {
				select {
				case stalled <- m:
				case <-ctx.Done():
					return
				}
			}
		}()

		protocols, _, err := newCluster(NewTranscoder(NewObserver(jrnl)), "server1")
		require.NoError(t, err)

		leader := findFirstState(raft.Leader, protocols...)
		sm := NewMachine(agent.NewPeer("node"), leader)

		start := time.Now()
		for i := 0; i < 50; i++ {
			require.NoError(t, sm.Dispatch(context.Background(), agent.LogInfo(local.Local(), "message")))
		}
		require.Less(t, time.Since(start), time.Second)
	})
}
