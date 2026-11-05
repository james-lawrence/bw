package raftutil_test

import (
	"context"
	"io"
	"log"
	"net"
	"path/filepath"

	"os"
	"testing"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/internal/md5x"
	"github.com/james-lawrence/bw/internal/testingx"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.SetOutput(io.Discard)
	code := m.Run()
	testingx.Cleanup()
	os.Exit(code)
}

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
