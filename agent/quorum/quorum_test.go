package quorum_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/storage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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

type stoppable interface {
	Stop()
}

func stop(servers ...*grpc.Server) {
	for _, n := range servers {
		n.Stop()
	}
}

func buildNode(p agent.Peer, deployer *mockdeploy) (wg *sync.WaitGroup, cancel context.CancelFunc, client agent.Client, err error) {
	var (
		l net.Listener
	)
	wg = &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	server := grpc.NewServer()
	c, err := agenttestutil.NewCluster(p)
	if err != nil {
		return wg, cancel, nil, err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		c.Leave()
		server.Stop()
		wg.Done()
	}()

	if l, err = agent.RPCTCPListener(c.Local()); err != nil {
		return wg, cancel, nil, err
	}

	wg.Add(1)
	go func() {
		<-ctx.Done()
		l.Close()
		wg.Done()
	}()

	quorum := New(
		c,
		deployer,
		storage.NewErrUploadProtocol(errors.New("upload protocol not specified")),
		OptionRaftProxy(NewRaftProxy(agent.RaftAddress(p), raft.Leader, nil)),
	)

	agent.RegisterAgentServer(server, agent.NewServer(c))
	agent.RegisterQuorumServer(server, agent.NewQuorum(&quorum))
	go server.Serve(l)
	client, err = agent.NewQuorumDialer(agent.NewDialer(grpc.WithInsecure())).Dial(c)
	return wg, cancel, client, err
}

var _ = Describe("Quorum", func() {
	Context("when leader", func() {
		It("should trigger a deploy", func() {
			md := &mockdeploy{}
			p := agent.NewPeer("127.0.0.1", agent.PeerOptionIP(net.ParseIP("127.0.0.1")))
			dopts := agent.DeployOptions{
				Timeout:     int64(time.Minute),
				Concurrency: 5,
			}
			wg, cancel, client, err := buildNode(p, md)
			defer wg.Wait()
			defer cancel()
			Expect(err).ToNot(HaveOccurred())
			archive := agent.Archive{DeploymentID: bw.MustGenerateID()}
			Expect(client.RemoteDeploy(dopts, archive)).ToNot(HaveOccurred())
			Expect(md.lastArchive).To(Equal(archive))
			Expect(md.lastOptions).To(Equal(dopts))
		})

		It("should be able to send and receive messages", func() {
			md := &mockdeploy{}
			p := agent.NewPeer("127.0.0.2", agent.PeerOptionIP(net.ParseIP("127.0.0.2")))
			wg, cancel, client, err := buildNode(p, md)
			defer wg.Wait()
			defer cancel()
			Expect(err).ToNot(HaveOccurred())

			dispatched := make(chan agent.Message, 100)
			go func() {
				client.Watch(dispatched)
			}()

			for i := 0; i < 10; i++ {
				msg := agentutil.LogEvent(agent.NewPeer("test"), "hello world")
				Expect(client.Dispatch(msg)).ToNot(HaveOccurred())
				Eventually(dispatched).Should(Receive(&msg))
			}
		})
	})
})
