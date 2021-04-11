package raftutil_test

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/internal/md5x"
	"github.com/james-lawrence/bw/internal/x/testingx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRaftutil(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.SetOutput(ioutil.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Raftutil Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)

type TestDialer struct {
	Dir string
}

// Dial is used to create a new outgoing connection
func (t TestDialer) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	host, _, _ := net.SplitHostPort(address)
	socket := filepath.Join(t.Dir, md5x.DigestString(host))

	d := &net.Dialer{}
	return d.DialContext(ctx, network, socket)
}

func SocketName(n *memberlist.Node) string {
	address := n.Addr.String()
	return md5x.DigestString(address)
}
