package raftutil_test

import (
	"context"
	"io"
	"log"
	"math/rand"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	. "github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/internal/testingx"
	"google.golang.org/grpc"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

func testRaftConfig() *raft.Config {
	conf := raft.DefaultConfig()

	conf.HeartbeatTimeout = 50 * time.Millisecond
	conf.ElectionTimeout = 50 * time.Millisecond
	conf.LeaderLeaseTimeout = 50 * time.Millisecond
	conf.CommitTimeout = 5 * time.Millisecond
	conf.SnapshotThreshold = 100
	conf.TrailingLogs = 10
	conf.LogOutput = io.Discard

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

func findPeer(r *raft.Raft, peers ...peer) *peer {
	for _, p := range peers {
		if strings.Contains(r.String(), SocketName(p.c.LocalNode())) {
			return &p
		}
	}
	return nil
}

func firstRaft(peers ...*raft.Raft) *raft.Raft {
	for _, p := range peers {
		return p
	}

	return nil
}

func findState(s raftStateFilter, peers ...*raft.Raft) []*raft.Raft {
	defer runtime.Gosched()

	r := make([]*raft.Raft, 0, len(peers))
	for _, p := range peers {
		if s(p.State()) {
			r = append(r, p)
		}
	}

	return r
}

func getServers(t *testing.T, rafts ...*raft.Raft) (ss []raft.Server) {
	for _, node := range findState(leaderFilter, rafts...) {
		config := node.GetConfiguration()
		require.NoError(t, config.Error())
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

//nolint:unused
func notShutdownFilter(i raft.RaftState) bool {
	return i != raft.Shutdown
}

//nolint:unused
func random(peers ...clustering.Memberlist) clustering.Memberlist {
	rand.Shuffle(len(peers), func(i int, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	for _, p := range peers {
		return p
	}

	panic("tried to get a random member of an empty set")
}

func overlayRaft(t *testing.T, ctx context.Context, q *grpc.ClientConn, tmpdir string, p clustering.Memberlist) (r Protocol) {
	var (
		err error
	)

	r, err = NewProtocol(
		ctx,
		p.LocalNode(),
		q,
		ProtocolOptionTransport(func() (raft.Transport, error) {
			var (
				l net.Listener
			)
			socket := filepath.Join(tmpdir, SocketName(p.LocalNode()))

			if l, err = net.Listen("unix", socket); err != nil {
				return nil, err
			}

			networkTransport := raft.NewNetworkTransportWithConfig(&raft.NetworkTransportConfig{
				Stream: NewStreamTransport(l, TestDialer{
					Dir: tmpdir,
				}),
				MaxPool: 1,
				Timeout: time.Second,
			})

			return networkTransport, nil
		}),
		ProtocolOptionConfig(testRaftConfig()),
		ProtocolOptionPassiveCheckin(200*time.Millisecond),
		ProtocolOptionLeadershipGrace(200*time.Millisecond),
	)
	require.NoError(t, err)
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

func newPeer(t *testing.T, ctx context.Context, tmpdir string, obs *raft.Observer, network *memberlist.MockNetwork, peers ...peer) peer {
	events := cluster.NewEventsQueue(nil)
	conn, _ := testingx.NewGRPCServer(t, func(srv *grpc.Server) {
		events.Bind(srv)
	})

	c, err := clusteringtestutil.NewPeer(network, clustering.OptionEventDelegate(events), clustering.OptionLogOutput(io.Discard))
	require.NoError(t, err)
	_, err = clusteringtestutil.Connect(c, clusters(peers...)...)
	require.NoError(t, err)
	rctx, rcancel := context.WithCancel(ctx)
	r := overlayRaft(t, rctx, conn, tmpdir, c)
	go r.Overlay(c, ProtocolOptionObservers(obs))
	return peer{c: c, r: &r, rc: rcancel}
}

func TestRaftPeers(t *testing.T) {
	t.Run("gracefully joins", func(t *testing.T) {
		var (
			network memberlist.MockNetwork
			rafts   []*raft.Raft
			peers   []peer
		)

		tmpdir := testingx.TempDir(t)
		obsc := make(chan raft.Observation, 100)
		obs := raft.NewObserver(obsc, true, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for i := 0; i < 3; i++ {
			peers = append(peers, newPeer(t, ctx, tmpdir, obs, &network, peers...))
		}

		require.Eventually(t, func() bool {
			rafts = gather(obsc, rafts...)
			return len(getServers(t, rafts...)) == 3
		}, 30*time.Second, 10*time.Millisecond)

		for i := 0; i < 5; i++ {
			peers = append(peers, newPeer(t, ctx, tmpdir, obs, &network, peers...))
			require.Eventually(t, func() bool {
				rafts = gather(obsc, rafts...)
				return len(getServers(t, rafts...)) >= 3
			}, 30*time.Second, 10*time.Millisecond)
		}
	})

	t.Run("should gracefully handle departures", func(t *testing.T) {
		var (
			network memberlist.MockNetwork
			rafts   []*raft.Raft
			peers   []peer
		)

		tmpdir := testingx.TempDir(t)
		obsc := make(chan raft.Observation, 100)
		obs := raft.NewObserver(obsc, true, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for i := 0; i < 5; i++ {
			peers = append(peers, newPeer(t, ctx, tmpdir, obs, &network, peers...))
		}

		require.Eventually(t, func() bool {
			rafts = gather(obsc, rafts...)
			return firstRaft(findState(leaderFilter, rafts...)...) != nil
		}, 10*time.Second, 10*time.Millisecond)

		leader := firstRaft(findState(leaderFilter, rafts...)...)
		require.NoError(t, leader.Barrier(time.Second).Error())

		for killed, i := 0, 0; i < len(peers) && killed < 2; i++ {
			p := peers[i]
			if strings.HasSuffix(string(leader.Leader()), p.c.LocalNode().Name) {
				continue
			}

			require.NoError(t, p.c.Shutdown())
			p.rc()
			killed++
		}

		require.Eventually(t, func() bool {
			rafts = gather(obsc, rafts...)
			return len(getServers(t, rafts...)) == 3
		}, 10*time.Second, 10*time.Millisecond)
	})

	t.Run("should allow for a single node to join and depart repeatedly", func(t *testing.T) {
		var (
			err     error
			network memberlist.MockNetwork
			rafts   []*raft.Raft
			peers   []peer
		)

		tmpdir := testingx.TempDir(t)
		obsc := make(chan raft.Observation, 100)
		obs := raft.NewObserver(obsc, true, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for i := 0; i < 3; i++ {
			peers = append(peers, newPeer(t, ctx, tmpdir, obs, &network, peers...))
		}

		require.Eventually(t, func() bool {
			rafts = gather(obsc, rafts...)
			return len(getServers(t, rafts...)) == 3
		}, 10*time.Second, 10*time.Millisecond)

		for i := 0; i < 5; i++ {
			var current *peer

			// pick a follower each iteration, the rejoined node may have become the leader.
			require.Eventually(t, func() bool {
				rafts = gather(obsc, rafts...)
				if len(voters(getServers(t, rafts...)...)) != 3 {
					return false
				}
				follower := firstRaft(findState(func(s raft.RaftState) bool { return s == raft.Follower }, rafts...)...)
				if follower == nil {
					return false
				}
				for j := range peers {
					if strings.Contains(follower.String(), SocketName(peers[j].c.LocalNode())) {
						current = &peers[j]
						return true
					}
				}
				return false
			}, 10*time.Second, 10*time.Millisecond)

			require.NoError(t, current.c.Shutdown())
			current.rc()

			require.Eventually(t, func() bool {
				rafts = gather(obsc, rafts...)
				return len(voters(getServers(t, rafts...)...)) == 2
			}, 10*time.Second, 10*time.Millisecond)

			// copy the config, the old memberlist may still be reading the original.
			cfg := *current.c.Config()
			current.c, err = clusteringtestutil.NewPeerFromConfig(&cfg, clustering.OptionNodeID(current.c.Config().Name))
			require.NoError(t, err)
			_, err = clusteringtestutil.Connect(current.c, clusters(peers...)...)
			require.NoError(t, err)
			rctx, rcancel := context.WithCancel(ctx)
			r := overlayRaft(t, rctx, current.r.StabilityQueue, tmpdir, current.c)
			go r.Overlay(current.c, ProtocolOptionObservers(obs))
			current.r, current.rc = &r, rcancel
		}
	})
}
