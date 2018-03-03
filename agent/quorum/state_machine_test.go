package quorum_test

import (
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

func (t *mockdeploy) Deploy(_ agent.Dialer, _ agent.Dispatcher, opts agent.DeployOptions, archive agent.Archive, peers ...agent.Peer) error {
	t.lastOptions = opts
	t.lastArchive = archive
	t.lastPeers = peers
	return nil
}

func newCluster(names ...string) (peers []*raft.Raft, err error) {
	var (
		servers    []raft.Server
		transports []*raft.InmemTransport
	)

	for _, n := range names {
		var (
			server    raft.Server
			transport *raft.InmemTransport
			protocol  *raft.Raft
		)

		if server, transport, protocol, err = newPeer(n, false); err != nil {
			return peers, err
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
			return peers, err
		}
	}

	return peers, nil
}

func newPeer(name string, leader bool) (raft.Server, *raft.InmemTransport, *raft.Raft, error) {
	var (
		addr      raft.ServerAddress
		transport *raft.InmemTransport
	)
	config := raft.DefaultConfig()
	config.StartAsLeader = leader
	config.LocalID = raft.ServerID(name)
	storage := raft.NewInmemStore()
	snapshot := raft.NewInmemSnapshotStore()
	addr, transport = raft.NewInmemTransport("")
	fsm := NewWAL()
	protocol, err := raft.NewRaft(config, &fsm, storage, storage, snapshot, transport)
	return raft.Server{Address: addr, ID: config.LocalID, Suffrage: raft.Voter}, transport, protocol, err
}

func connect(local *raft.InmemTransport, peers ...*raft.InmemTransport) {
	for _, p := range peers {
		local.Connect(p.LocalAddr(), p)
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
		protocols, err := newCluster("server1", "server2", "server3")
		Expect(err).ToNot(HaveOccurred())
		log.Println("awaiting leader")
		leader := awaitLeader(protocols...)
		log.Println("leader elected")
		lp := agent.NewPeer("node")
		mock := clustering.NewMock(clusteringtestutil.NewNode(lp.Name, net.ParseIP(lp.Ip)))

		sm := NewStateMachine(cluster.New(cluster.NewLocal(lp), mock), leader, agent.NewDialer(), &mockdeploy{})
		cmd := qCommand(agent.DeployCommand_Begin)

		Expect((&sm).Dispatch(agentutil.DeployCommand(lp, cmd))).ToNot(HaveOccurred())
	})

	It("should write message to the observer", func() {
		protocols, err := newCluster("server1", "server2", "server3")
		Expect(err).ToNot(HaveOccurred())
		leader := awaitLeader(protocols...)
		messages := []agent.Message{
			agentutil.LogEvent(local.Local(), "message 1"),
			agentutil.LogEvent(local.Local(), "message 2"),
			agentutil.LogEvent(local.Local(), "message 3"),
			agentutil.LogEvent(local.Local(), "message 4"),
		}
		obs := make(chan agent.Message, len(messages))
		lp := agent.NewPeer("node")
		mock := clustering.NewMock(clusteringtestutil.NewNode(lp.Name, net.ParseIP(lp.Ip)))
		sm := NewStateMachine(cluster.New(cluster.NewLocal(lp), mock), leader, agent.NewDialer(), &mockdeploy{}, STOObserver(obs))

		Expect((&sm).Dispatch(messages...)).ToNot(HaveOccurred())

		for _, m := range messages {
			var expected agent.Message
			Eventually(obs).Should(Receive(&expected))
			Expect(expected).To(Equal(m))
		}
	})

	DescribeTable("persisting state",
		func(n int, messages ...agent.Message) {
			var (
				decoded agent.WAL
			)
			convert := func(in ...agent.Message) (out []*agent.Message) {
				for _, m := range in {
					tmp := m
					out = append(out, &tmp)
				}
				return out
			}
			protocols, err := newCluster("server1", "server2", "server3")
			Expect(err).ToNot(HaveOccurred())
			leader := awaitLeader(protocols...)
			lp := agent.NewPeer("node")
			mock := clustering.NewMock(clusteringtestutil.NewNode(lp.Name, net.ParseIP(lp.Ip)))
			sm := NewStateMachine(cluster.New(cluster.NewLocal(lp), mock), leader, agent.NewDialer(), &mockdeploy{})

			Expect((&sm).Dispatch(messages...)).ToNot(HaveOccurred())
			snapshotfuture := leader.Snapshot()
			Expect(snapshotfuture.Error()).ToNot(HaveOccurred())
			_, ior, err := snapshotfuture.Open()
			Expect(err).ToNot(HaveOccurred())
			raw, err := ioutil.ReadAll(ior)
			Expect(err).ToNot(HaveOccurred())
			Expect(proto.Unmarshal(raw, &decoded)).ToNot(HaveOccurred())
			Expect(decoded.Messages).To(ConsistOf(convert(messages...)[n:]))
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
	)
})
