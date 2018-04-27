package observers_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agent/observers"
	"github.com/james-lawrence/bw/agentutil"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Observers", func() {
	It("should be able to receive messages", func() {
		dir, err := ioutil.TempDir(".", "observers")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(dir)
		addr := filepath.Join(dir, fmt.Sprintf("%s.sock", bw.MustGenerateID().String()))
		l, err := net.Listen("unix", addr)
		Expect(err).ToNot(HaveOccurred())
		defer l.Close()
		dst := make(chan agent.Message, 10)
		gs := New(dst)
		go gs.Serve(l)

		obsc, err := NewDialer(context.Background(), addr, grpc.WithInsecure(), grpc.WithBlock())
		Expect(err).ToNot(HaveOccurred())
		Expect(
			obsc.Dispatch(
				context.Background(),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 0"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 1"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 2"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 3"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 4"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 5"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 6"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 7"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 8"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 9"),
			),
		).ToNot(HaveOccurred())

		// 2nd batch, should time out due to a full buffer.
		ctx, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		Expect(
			obsc.Dispatch(
				ctx,
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 10"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 11"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 12"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 13"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 14"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 15"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 16"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 17"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 18"),
				agentutil.LogEvent(agent.NewPeer("peer1"), "hello world 19"),
			),
		).To(HaveOccurred())

		Expect(len(dst)).To(Equal(10))
	})
})
