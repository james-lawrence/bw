package raftutil_test

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/clusteringtestutil"
	"bitbucket.org/jatone/bearded-wookie/clustering/peering"
	. "bitbucket.org/jatone/bearded-wookie/clustering/raftutil"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/icrowley/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testRaftConfig() *raft.Config {
	conf := raft.DefaultConfig()

	conf.HeartbeatTimeout = 50 * time.Millisecond
	conf.ElectionTimeout = 50 * time.Millisecond
	conf.LeaderLeaseTimeout = 50 * time.Millisecond
	conf.CommitTimeout = 5 * time.Millisecond
	conf.SnapshotThreshold = 100
	conf.TrailingLogs = 10
	conf.LogOutput = ioutil.Discard

	return conf
}

var _ = Describe("Raft", func() {
	Context("peers", func() {
		It("a node should be able to join and leave repeatedly", func() {

			newLocalPeer := func(p *memberlist.Node) cluster.Local {
				return cluster.NewLocal(agent.Peer{
					Ip:       p.Addr.String(),
					Name:     p.Name,
					RPCPort:  uint32(2000),
					SWIMPort: uint32(2001),
					RaftPort: uint32(2002),
				})
			}

			newLocalNode := func(l cluster.Local) (cancel context.CancelFunc, c clustering.Cluster, r *Protocol, err error) {
				var (
					_r Protocol
				)
				ctx, cancel := context.WithCancel(context.Background())
				_r, err = NewProtocol(
					ctx,
					uint16(l.Peer.RaftPort),
					ProtocolOptionTransport(func() (raft.Transport, error) {
						tcp := &net.TCPAddr{IP: net.ParseIP(l.Peer.Ip), Port: int(l.Peer.RaftPort)}
						return raft.NewTCPTransport(tcp.String(), nil, 3, time.Second, os.Stderr)
					}),
					ProtocolOptionSnapshotStorage(raft.NewInmemSnapshotStore()),
					ProtocolOptionConfig(testRaftConfig()),
				)

				if err != nil {
					return cancel, c, r, err
				}

				r = &_r
				c, err = clustering.NewOptionsFromConfig(
					memberlist.DefaultLocalConfig(),
					clustering.OptionNodeID(l.Peer.Name),
					clustering.OptionDelegate(l),
					clustering.OptionEventDelegate(r),
					clustering.OptionBindAddress(l.Peer.Ip),
					clustering.OptionBindPort(int(l.Peer.SWIMPort)),
					clustering.OptionLogOutput(ioutil.Discard),
				).NewCluster()
				if err != nil {
					return cancel, c, r, err
				}

				go r.Overlay(c)
				return cancel, c, r, nil
			}
			p1 := clusteringtestutil.NewPeer(fake.CharactersN(10), net.ParseIP("127.0.0.1"))
			lp1 := newLocalPeer(p1)
			cancel1, c1, r1, err := newLocalNode(lp1)
			Expect(err).ToNot(HaveOccurred())
			p2 := clusteringtestutil.NewPeer(fake.CharactersN(10), net.ParseIP("127.0.0.2"))
			lp2 := newLocalPeer(p2)
			cancel2, c2, r2, err := newLocalNode(lp2)
			defer cancel2()
			Expect(err).ToNot(HaveOccurred())
			p3 := clusteringtestutil.NewPeer(fake.CharactersN(10), net.ParseIP("127.0.0.3"))
			lp3 := newLocalPeer(p3)
			cancel3, c3, r3, err := newLocalNode(lp3)
			defer cancel3()
			Expect(err).ToNot(HaveOccurred())

			log.Println("peers1", clustering.Peers(c1))
			Expect(clustering.Bootstrap(
				c2,
				clustering.BootstrapOptionPeeringStrategies(
					peering.NewStatic(clustering.Peers(c1)...),
				),
			)).ToNot(HaveOccurred())

			log.Println("peers2", clustering.Peers(c1))
			Expect(clustering.Bootstrap(
				c3,
				clustering.BootstrapOptionPeeringStrategies(
					peering.NewStatic(clustering.Peers(c1)...),
				),
			)).ToNot(HaveOccurred())
			log.Println("peers3", clustering.Peers(c1))

			Eventually(func() bool { return r1.Raft() == nil }).ShouldNot(BeTrue())
			Eventually(func() bool { return r1.Raft().Leader() == "" }, 10*time.Second).ShouldNot(BeTrue())
			Eventually(func() bool { return r2.Raft().Leader() == "" }, 10*time.Second).ShouldNot(BeTrue())
			Eventually(func() bool { return r3.Raft().Leader() == "" }, 10*time.Second).ShouldNot(BeTrue())

			for i := 0; i < 5; i++ {
				log.Println("-------------- ATTEMPTING SHUTDOWN --------------", i, r1.Raft().State(), r2.Raft().State(), r3.Raft().State())
				Expect(c1.Shutdown()).ToNot(HaveOccurred())
				cancel1()

				log.Println("-------------- SUCCESS SHUTDOWN --------------", i, r1.Raft().State())

				cancel1, c1, r1, err = newLocalNode(lp1)
				Expect(err).ToNot(HaveOccurred())
				Expect(clustering.Bootstrap(
					c1,
					clustering.BootstrapOptionPeeringStrategies(
						peering.NewStatic(clustering.Peers(c2)...),
					),
				)).ToNot(HaveOccurred())

				Eventually(func() bool { r1.ClusterChange.Broadcast(); return r1.Raft() == nil }, 10*time.Second).ShouldNot(BeTrue())
				Eventually(func() bool { return r1.Raft().Leader() == "" }, 10*time.Second).ShouldNot(BeTrue())
				Eventually(func() bool { return r2.Raft().Leader() == "" }, 10*time.Second).ShouldNot(BeTrue())
				Eventually(func() bool { return r3.Raft().Leader() == "" }, 10*time.Second).ShouldNot(BeTrue())
			}
		})
	})
})
