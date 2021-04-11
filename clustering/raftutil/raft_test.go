package raftutil_test

import (
	"context"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	. "github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/internal/x/testingx"
	"google.golang.org/grpc"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
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

func gather(c chan raft.Observation, peers ...*raft.Raft) (results []*raft.Raft) {
	seen := map[*raft.Raft]bool{}
	for _, p := range peers {
		seen[p] = true
	}
	results = peers
	for {
		select {
		case o := <-c:
			if ok := seen[o.Raft]; !ok {
				seen[o.Raft] = true
				results = append(results, o.Raft)
			}
		default:
			return results
		}
	}
}

func printrafts(peers ...*raft.Raft) {
	log.Println("checkpoint initiated")
	defer log.Println("checkpoint completed")
	for _, p := range peers {
		log.Println(p.String())
	}
}

type raftStateFilter func(raft.RaftState) bool

func firstRaft(peers ...*raft.Raft) *raft.Raft {
	for _, p := range peers {
		return p
	}

	return nil
}

func findState(s raftStateFilter, peers ...*raft.Raft) []*raft.Raft {
	r := make([]*raft.Raft, 0, len(peers))
	for _, p := range peers {
		if s(p.State()) {
			r = append(r, p)
		}
	}

	return r
}

func getServers(rafts ...*raft.Raft) (ss []raft.Server) {
	for _, node := range findState(leaderFilter, rafts...) {
		config := node.GetConfiguration()
		Expect(config.Error()).ToNot(HaveOccurred())
		return config.Configuration().Servers
	}
	return []raft.Server{}
}

func voters(sss ...raft.Server) (ss []raft.Server) {
	for _, s := range sss {
		if s.Suffrage == raft.Voter {
			ss = append(ss, s)
		}
	}
	return ss
}

func leaderFilter(i raft.RaftState) bool {
	return i == raft.Leader
}

func notLeader(i raft.RaftState) bool {
	return i != raft.Leader
}

func notShutdownFilter(i raft.RaftState) bool {
	return i != raft.Shutdown
}

func random(peers ...clustering.Memberlist) clustering.Memberlist {
	rand.Shuffle(len(peers), func(i int, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	for _, p := range peers {
		return p
	}

	panic("tried to get a random member of an empty set")
}

func overlayRaft(ctx context.Context, q BacklogQueueWorker, tmpdir string, p clustering.Memberlist) (r Protocol) {
	var (
		err error
	)

	r, err = NewProtocol(
		ctx,
		q,
		ProtocolOptionTransport(func() (raft.Transport, error) {
			var (
				l net.Listener
			)

			if l, err = net.Listen("unix", filepath.Join(tmpdir, p.LocalNode().Name)); err != nil {
				return nil, err
			}

			networkTransport := raft.NewNetworkTransportWithConfig(&raft.NetworkTransportConfig{
				// Logger:  log.New(ioutil.Discard, "", log.LstdFlags),
				Stream:  NewUnixStreamLayer(l),
				MaxPool: 1,
				Timeout: time.Second,
			})

			return networkTransport, nil
		}),
		ProtocolOptionConfig(testRaftConfig()),
		ProtocolOptionEnableSingleNode(false),
		ProtocolOptionPassiveCheckin(200*time.Millisecond),
		ProtocolOptionLeadershipGrace(200*time.Millisecond),
	)
	Expect(err).ToNot(HaveOccurred())
	return r
}

type peer struct {
	c  clustering.Memberlist
	r  *Protocol
	rc context.CancelFunc
}

func clusters(peers ...peer) (o []clustering.Memberlist) {
	for _, p := range peers {
		o = append(o, p.c)
	}

	return o
}

func newPeer(ctx context.Context, tmpdir string, obs *raft.Observer, network *memberlist.MockNetwork, peers ...peer) peer {
	events := cluster.NewEventsQueue(nil)
	conn, _ := testingx.NewGRPCServer(func(srv *grpc.Server) {
		events.Bind(srv)
	})

	sq := BacklogQueueWorker{Queue: make(chan *agent.ClusterWatchEvents)}

	Expect(
		cluster.NewEventsSubscription(conn, sq.Enqueue),
	).To(Succeed())

	c, err := clusteringtestutil.NewPeer(network, clustering.OptionEventDelegate(events), clustering.OptionLogOutput(ioutil.Discard))
	Expect(err).ToNot(HaveOccurred())
	_, err = clusteringtestutil.Connect(c, clusters(peers...)...)
	Expect(err).ToNot(HaveOccurred())
	rctx, rcancel := context.WithCancel(ctx)
	r := overlayRaft(rctx, sq, tmpdir, c)
	go r.Overlay(c, ProtocolOptionObservers(obs))
	return peer{c: c, r: &r, rc: rcancel}
}

var _ = Describe("Raft", func() {
	Context("peers", func() {
		It("gracefully joins", func() {
			var (
				network memberlist.MockNetwork
				rafts   []*raft.Raft
				peers   []peer
			)

			tmpdir := testingx.TempDir()
			obsc := make(chan raft.Observation, 100)
			obs := raft.NewObserver(obsc, true, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			for i := 0; i < 3; i++ {
				peers = append(peers, newPeer(ctx, tmpdir, obs, &network, peers...))
			}

			Eventually(func() []raft.Server {
				rafts = gather(obsc, rafts...)
				return getServers(rafts...)
			}, 30*time.Second).Should(HaveLen(3))

			for i := 0; i < 5; i++ {
				peers = append(peers, newPeer(ctx, tmpdir, obs, &network, peers...))
				Eventually(func() []raft.Server {
					rafts = gather(obsc, rafts...)
					return getServers(rafts...)
				}, 30*time.Second).Should(HaveLen(3))
			}
		})

		It("should gracefully handle departures", func() {
			var (
				network memberlist.MockNetwork
				rafts   []*raft.Raft
				peers   []peer
			)

			tmpdir := testingx.TempDir()
			obsc := make(chan raft.Observation, 100)
			obs := raft.NewObserver(obsc, true, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			for i := 0; i < 5; i++ {
				peers = append(peers, newPeer(ctx, tmpdir, obs, &network, peers...))
			}

			Eventually(func() *raft.Raft {
				rafts = gather(obsc, rafts...)
				return firstRaft(findState(leaderFilter, rafts...)...)
			}).ShouldNot(BeNil())

			leader := firstRaft(findState(leaderFilter, rafts...)...)

			for killed, i := 0, 0; i < len(peers) && killed < 2; i++ {
				p := peers[i]
				if strings.HasSuffix(string(leader.Leader()), p.c.LocalNode().Name) {
					continue
				}

				Expect(p.c.Shutdown()).ToNot(HaveOccurred())
				p.rc()
				killed++
			}

			Eventually(func() []raft.Server {
				rafts = gather(obsc, rafts...)
				return getServers(rafts...)
			}, 30*time.Second).Should(HaveLen(3))
		})

		It("should allow for a single node to join and depart repeatedly", func() {
			var (
				err     error
				network memberlist.MockNetwork
				rafts   []*raft.Raft
				peers   []peer
			)

			tmpdir := testingx.TempDir()
			obsc := make(chan raft.Observation, 100)
			obs := raft.NewObserver(obsc, true, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			for i := 0; i < 2; i++ {
				peers = append(peers, newPeer(ctx, tmpdir, obs, &network, peers...))
			}

			Eventually(func() []raft.Server {
				rafts = gather(obsc, rafts...)
				return getServers(rafts...)
			}, 30*time.Second).Should(HaveLen(2))

			peer := newPeer(ctx, tmpdir, obs, &network, peers...)
			for i := 0; i < 5; i++ {
				Eventually(func() []raft.Server {
					rafts = gather(obsc, rafts...)
					return voters(getServers(rafts...)...)
				}, 30*time.Second).Should(HaveLen(3))

				Expect(peer.c.Shutdown()).ToNot(HaveOccurred())
				peer.rc()

				Eventually(func() []raft.Server {
					rafts = gather(obsc, rafts...)
					return voters(getServers(rafts...)...)
				}, 30*time.Second).Should(HaveLen(2))

				peer.c, err = clusteringtestutil.NewPeerFromConfig(peer.c.Config(), clustering.OptionNodeID(peer.c.Config().Name))
				Expect(err).ToNot(HaveOccurred())
				_, err = clusteringtestutil.Connect(peer.c, clusters(peers...)...)
				Expect(err).ToNot(HaveOccurred())
				rctx, rcancel := context.WithCancel(ctx)
				r := overlayRaft(rctx, peer.r.StabilityQueue, tmpdir, peer.c)
				go r.Overlay(peer.c, ProtocolOptionObservers(obs))
				peer.r, peer.rc = &r, rcancel
			}
		})

		It("should handle leadership changes", func() {
			var (
				network memberlist.MockNetwork
				rafts   []*raft.Raft
				peers   []peer
			)

			tmpdir := testingx.TempDir()
			obsc := make(chan raft.Observation, 100)
			obs := raft.NewObserver(obsc, true, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			for i := 0; i < 3; i++ {
				peers = append(peers, newPeer(ctx, tmpdir, obs, &network, peers...))
			}

			Eventually(func() *raft.Raft {
				rafts = gather(obsc, rafts...)
				return firstRaft(findState(leaderFilter, rafts...)...)
			}).ShouldNot(BeNil())

			leader := firstRaft(findState(leaderFilter, rafts...)...)
			expected := leader.Leader()

			Eventually(func() bool {
				newPeer(ctx, tmpdir, obs, &network, peers...)
				time.Sleep(500 * time.Millisecond)
				rafts = gather(obsc, rafts...)
				if l2 := firstRaft(findState(leaderFilter, rafts...)...); l2 != nil && l2.Leader() != expected {
					return true
				}
				return false
			}, 30*time.Second).Should(BeTrue())

			Eventually(func() *raft.Raft {
				time.Sleep(200 * time.Millisecond)
				rafts = gather(obsc, rafts...)
				return firstRaft(findState(leaderFilter, rafts...)...)
			}, 30*time.Second).ShouldNot(BeNil())

			Eventually(func() bool {
				time.Sleep(200 * time.Millisecond)
				rafts = gather(obsc, rafts...)
				if l2 := firstRaft(findState(leaderFilter, rafts...)...); l2 != nil {
					log.Println("VERIFYING NEW LEADER", l2.Leader(), expected)
					return l2.Leader() != expected
				}
				return false
			}, 30*time.Second).Should(BeTrue())
		})
	})
})
