package agent_test

import (
	"context"
	"io"
	"net"

	. "github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/icrowley/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testauth struct{}

func (t testauth) Deploy(ctx context.Context) error {
	return nil
}

type harness struct {
	client   Client
	cluster  cluster.Cluster
	listener net.Listener
}

func (t harness) Cleanup() {
	Expect(t.listener.Close()).ToNot(HaveOccurred())
}

func testClient() harness {
	socket, err := net.Listen("tcp", ":0")
	Expect(err).ToNot(HaveOccurred())
	peers := clusteringtestutil.NewNodes(5)

	c := cluster.New(
		NewPeer(fake.CharactersN(10)),
		clustering.NewMock(peers[0], peers[1:]...),
	)
	s := NewServer(c, ServerOptionAuth(testauth{}))

	grpcs := grpc.NewServer()
	RegisterAgentServer(grpcs, s)
	go func() {
		grpcs.Serve(socket)
	}()

	conn, err := Dial(socket.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	Expect(err).ToNot(HaveOccurred())

	return harness{client: conn, cluster: c, listener: socket}
}

var _ = Describe("Server", func() {
	Context("Connect", func() {
		It("should return cluster details", func() {
			h := testClient()
			defer h.Cleanup()

			q := h.cluster.Quorum()
			info, err := h.client.Connect(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Quorum).To(ConsistOf(q[0], q[1], q[2]))
		})
	})

	Context("Info", func() {
		It("should return information about the agent", func() {
			h := testClient()
			defer h.Cleanup()

			info, err := h.client.Info(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Deployments).To(BeEmpty())
			tmp := h.cluster.Local()
			tmp.Status = Peer_Node
			Expect(proto.Equal(info.Peer, tmp)).To(BeTrue())
		})
	})

	Context("Deploy", func() {
		It("should trigger a deploy on the server", func() {
			h := testClient()
			defer h.Cleanup()

			_, err := h.client.Deploy(context.Background(), &DeployOptions{}, &Archive{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Logs", func() {
		It("should return logs from the given deployer", func() {
			h := testClient()
			defer h.Cleanup()
			p := h.cluster.Local()
			pipe := h.client.Logs(context.Background(), p, []byte("fake"))
			raw, err := io.ReadAll(pipe)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(raw)).To(Equal("INFO: fake"))
		})
	})
})
