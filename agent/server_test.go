package agent_test

import (
	"context"

	. "github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"

	"github.com/icrowley/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	Context("Connect", func() {
		It("should return cluster details", func() {
			peers := clusteringtestutil.NewPeers(5)

			c := cluster.New(
				cluster.NewLocal(NewPeer(fake.CharactersN(10))),
				clustering.NewMock(peers[0], peers[1:]...),
			)
			q := PeersToPtr(c.Quorum()...)
			s := NewServer(c)
			info, err := s.Connect(context.Background(), &ConnectRequest{})
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Quorum).To(ConsistOf(q[0], q[1], q[2]))
		})
	})

	Context("Info", func() {
		It("should return information about the agent", func() {
			peers := clusteringtestutil.NewPeers(5)
			c := cluster.New(
				cluster.NewLocal(NewPeer(fake.CharactersN(10))),
				clustering.NewMock(peers[0], peers[1:]...),
			)
			s := NewServer(c)
			info, err := s.Info(context.Background(), &StatusRequest{})
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Deployments).To(BeEmpty())
			tmp := c.Local()
			tmp.Status = Peer_Node
			Expect(info.Peer).To(Equal(&tmp))
		})
	})

	Context("Deploy", func() {
		It("should trigger a deploy on the server", func() {
			peers := clusteringtestutil.NewPeers(5)
			c := cluster.New(
				cluster.NewLocal(NewPeer(fake.CharactersN(10))),
				clustering.NewMock(peers[0], peers[1:]...),
			)
			s := NewServer(c)
			_, err := s.Deploy(context.Background(), &Archive{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
