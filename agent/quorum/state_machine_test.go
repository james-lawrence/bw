package quorum_test

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"runtime"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type mockLocal struct{}

func (mockLocal) Local() agent.Peer {
	return agent.NewPeer("")
}

type mockdeploy struct {
	lastOptions agent.DeployOptions
	lastArchive agent.Archive
	lastPeers   []agent.Peer
}

func (t *mockdeploy) Deploy(_ agent.Dialer, opts agent.DeployOptions, archive agent.Archive, peers ...agent.Peer) error {
	t.lastOptions = opts
	t.lastArchive = archive
	t.lastPeers = peers
	return nil
}

func newCluster(obs chan agent.Message, names ...string) (peers []*raft.Raft, transports []*raft.InmemTransport, err error) {
	var (
		servers []raft.Server
	)

	for _, n := range names {
		var (
			server    raft.Server
			transport *raft.InmemTransport
			protocol  *raft.Raft
		)

		if server, transport, protocol, err = newPeer(obs, n, false); err != nil {
			return peers, transports, err
		}
		// don't add the observer to any other nodes.
		obs = nil
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

func newPeer(obs chan agent.Message, name string, leader bool) (raft.Server, *raft.InmemTransport, *raft.Raft, error) {
	var (
		addr      raft.ServerAddress
		transport *raft.InmemTransport
	)
	config := raft.DefaultConfig()
	config.LogOutput = ioutil.Discard
	config.StartAsLeader = leader
	config.LocalID = raft.ServerID(name)
	storage := raft.NewInmemStore()
	snapshot := raft.NewInmemSnapshotStore()
	addr, transport = raft.NewInmemTransport("")
	fsm := NewWAL(obs)
	protocol, err := raft.NewRaft(config, &fsm, storage, storage, snapshot, transport)
	return raft.Server{Address: addr, ID: config.LocalID, Suffrage: raft.Voter}, transport, protocol, err
}

func connect(local *raft.InmemTransport, peers ...*raft.InmemTransport) {
	for _, p := range peers {
		local.Connect(p.LocalAddr(), p)
		p.Connect(local.LocalAddr(), local)
	}
}

func awaitLeader(protocols ...*raft.Raft) *raft.Raft {
	for {
		for _, p := range protocols {
			if p.State() == raft.Leader {
				return p
			}
		}
		runtime.Gosched()
	}
}

func awaitFollower(protocols ...*raft.Raft) *raft.Raft {
	for {
		for _, p := range protocols {
			if p.State() == raft.Follower {
				return p
			}
		}
		runtime.Gosched()
	}
}

func qCommand(d agent.DeployCommand_Command) agent.DeployCommand {
	return agent.DeployCommand{
		Command: d,
		Archive: &agent.Archive{},
		Options: &agent.DeployOptions{},
	}
}

var _ = Describe("StateMachine", func() {
	local := mockLocal{}

	It("should write to WAL on dispatch", func() {
		protocols, _, err := newCluster(nil, "server1", "server2", "server3")
		Expect(err).ToNot(HaveOccurred())
		log.Println("awaiting leader")
		leader := awaitLeader(protocols...)
		log.Println("leader elected")
		lp := agent.NewPeer("node")
		mock := clustering.NewMock(clusteringtestutil.NewNode(lp.Name, net.ParseIP(lp.Ip)))
		wal := NewWAL(make(chan agent.Message))
		sm := NewStateMachine(&wal, cluster.New(cluster.NewLocal(lp), mock), leader, agent.NewDialer(), &mockdeploy{})
		cmd := qCommand(agent.DeployCommand_Begin)

		Expect((&sm).Dispatch(context.Background(), agentutil.DeployCommand(lp, cmd))).ToNot(HaveOccurred())
	})

	DescribeTable("persisting state",
		func(n int, messages ...agent.Message) {
			var (
				decoded agent.WAL
			)
			convert := func(c []*agent.Message) []interface{} {
				r := make([]interface{}, 0, len(c))
				for _, v := range c {
					v.Replay = true
					r = append(r, v)
				}
				return r
			}
			protocols, _, err := newCluster(nil, "server1", "server2", "server3")
			Expect(err).ToNot(HaveOccurred())
			leader := awaitLeader(protocols...)
			lp := agent.NewPeer("node")
			mock := clustering.NewMock(clusteringtestutil.NewNode(lp.Name, net.ParseIP(lp.Ip)))
			wal := NewWAL(make(chan agent.Message))
			sm := NewStateMachine(&wal, cluster.New(cluster.NewLocal(lp), mock), leader, agent.NewDialer(), &mockdeploy{})

			Expect((&sm).Dispatch(context.Background(), messages...)).ToNot(HaveOccurred())
			snapshotfuture := leader.Snapshot()
			Expect(snapshotfuture.Error()).ToNot(HaveOccurred())
			_, ior, err := snapshotfuture.Open()
			Expect(err).ToNot(HaveOccurred())
			raw, err := ioutil.ReadAll(ior)
			Expect(err).ToNot(HaveOccurred())
			Expect(proto.Unmarshal(raw, &decoded)).ToNot(HaveOccurred())
			Expect(decoded.Messages).To(ConsistOf(convert(agent.MessagesToPtr(messages[n:]...))...))
		},
		Entry(
			"example 1",
			4,
			agentutil.LogEvent(local.Local(), "message 1"),
			agentutil.LogEvent(local.Local(), "message 2"),
			agentutil.LogEvent(local.Local(), "message 3"),
			agentutil.LogEvent(local.Local(), "message 4"),
			agentutil.DeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
			agentutil.LogEvent(local.Local(), "message 5"),
			agentutil.LogEvent(local.Local(), "message 6"),
		),
		Entry(
			"example 2",
			6,
			agentutil.LogEvent(local.Local(), "message 1"),
			agentutil.LogEvent(local.Local(), "message 2"),
			agentutil.DeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
			agentutil.LogEvent(local.Local(), "message 3"),
			agentutil.DeployCommand(local.Local(), qCommand(agent.DeployCommand_Done)),
			agentutil.LogEvent(local.Local(), "message 4"),
			agentutil.DeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
			agentutil.LogEvent(local.Local(), "message 5"),
			agentutil.LogEvent(local.Local(), "message 6"),
		),
		Entry(
			"example 3",
			0,
			agentutil.LogEvent(local.Local(), "message 1"),
			agentutil.LogEvent(local.Local(), "message 2"),
		),
		Entry(
			"example 4",
			0,
			agentutil.DeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
		),
		Entry(
			"example 5",
			7,
			agentutil.LogEvent(local.Local(), "message 1"),
			agentutil.LogEvent(local.Local(), "message 2"),
			agentutil.LogEvent(local.Local(), "message 3"),
			agentutil.LogEvent(local.Local(), "message 4"),
			agentutil.DeployCommand(local.Local(), qCommand(agent.DeployCommand_Begin)),
			agentutil.LogEvent(local.Local(), "message 5"),
			agentutil.LogEvent(local.Local(), "message 6"),
			agentutil.DeployCommand(local.Local(), qCommand(agent.DeployCommand_Done)),
		),
	)

	It("should return an error when dispatch failed", func() {
		cmd := agent.DeployCommand{
			Command: agent.DeployCommand_Begin,
			Options: &agent.DeployOptions{},
			Archive: &agent.Archive{},
		}

		protocols, _, err := newCluster(nil, "server1", "server2", "server3")
		Expect(err).ToNot(HaveOccurred())
		leader := awaitLeader(protocols...)
		lp := agent.NewPeer("node")
		mock := clustering.NewMock(clusteringtestutil.NewNode(lp.Name, net.ParseIP(lp.Ip)))
		wal := NewWAL(make(chan agent.Message))
		sm := NewStateMachine(&wal, cluster.New(cluster.NewLocal(lp), mock), leader, agent.NewDialer(), &mockdeploy{})

		Expect((&sm).Dispatch(context.Background(), agentutil.DeployCommand(local.Local(), cmd))).ToNot(HaveOccurred())
		Expect((&sm).Dispatch(context.Background(), agentutil.DeployCommand(local.Local(), cmd))).To(HaveOccurred())
	})

	It("should write message to the observer", func() {
		messages := []agent.Message{
			agentutil.LogEvent(local.Local(), "message 1"),
			agentutil.LogEvent(local.Local(), "message 2"),
			agentutil.LogEvent(local.Local(), "message 3"),
			agentutil.LogEvent(local.Local(), "message 4"),
		}
		obs := make(chan agent.Message, len(messages))
		protocols, _, err := newCluster(obs, "server1", "server2", "server3")
		Expect(err).ToNot(HaveOccurred())
		leader := awaitLeader(protocols...)
		lp := agent.NewPeer("node")
		mock := clustering.NewMock(clusteringtestutil.NewNode(lp.Name, net.ParseIP(lp.Ip)))
		wal := NewWAL(make(chan agent.Message))
		sm := NewStateMachine(&wal, cluster.New(cluster.NewLocal(lp), mock), leader, agent.NewDialer(), &mockdeploy{})

		Expect((&sm).Dispatch(context.Background(), messages...)).ToNot(HaveOccurred())

		for _, m := range messages {
			var expected agent.Message
			Eventually(obs).Should(Receive(&expected))
			Expect(expected).To(Equal(m))
		}
	})
})
