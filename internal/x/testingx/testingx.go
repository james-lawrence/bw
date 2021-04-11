package testingx

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/akutz/memconn"
	"github.com/gofrs/uuid"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
)

const rootDir = ".tests"

// TempDir generates a tmp directory within the root testing directory for use in tests.
func TempDir() (dir string) {
	var err error
	setup()

	if err = os.MkdirAll(rootDir, 0755); err != nil {
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	if dir, err = ioutil.TempDir(rootDir, "tmp"); err != nil {
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	return dir
}

// Cleanup used to cleanup the test directory after a suite is run.
func Cleanup() {
	gomega.Expect(os.RemoveAll(rootDir)).ToNot(gomega.HaveOccurred())
}

// NewGRPCServer sets up a server and a connection.
func NewGRPCServer(bind func(s *grpc.Server), options ...grpc.DialOption) (c *grpc.ClientConn, s *grpc.Server) {
	inmemconn := uuid.Must(uuid.NewV4()).String()
	l, err := memconn.Listen("memu", inmemconn)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	s = grpc.NewServer()
	bind(s)
	go func() {
		s.Serve(l)
	}()

	options = append([]grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithContextDialer(func(ctx context.Context, address string) (conn net.Conn, err error) {
			ctx, done := context.WithTimeout(ctx, time.Second)
			defer done()
			return memconn.DialContext(ctx, "memu", address)
		}),
	}, options...)
	c, err = grpc.Dial(l.Addr().String(), options...)

	gomega.Expect(err).To(gomega.Succeed())

	return c, s
}

// NewGRPCServer sets up a server and a dialer.
func NewGRPCServer2(bind func(s *grpc.Server), options ...grpc.DialOption) (d dialers.Direct, s *grpc.Server) {
	l, err := net.Listen("tcp", ":0")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	s = grpc.NewServer()
	bind(s)
	go func() {
		s.Serve(l)
	}()

	options = append(
		options,
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, to time.Duration) (c net.Conn, err error) {
			ctx, done := context.WithTimeout(context.Background(), to)
			defer done()
			return (&net.Dialer{}).DialContext(ctx, l.Addr().Network(), l.Addr().String())
		}),
	)
	return dialers.NewDirect(l.Addr().String(), dialers.NewDefaults(options...).Defaults()...), s
}

// GRPCCleanup shuts down the client and server. for use with defer.
func GRPCCleanup(c *grpc.ClientConn, s *grpc.Server) {
	if c != nil {
		gomega.Expect(c.Close()).ToNot(gomega.HaveOccurred())
	}
	s.Stop()
}

func setup() {
	var (
		err error
	)

	if err = os.MkdirAll(rootDir, 0755); err != nil {
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func NewCachedDialer(c *grpc.ClientConn) CachedDialer {
	return CachedDialer{ClientConn: c}
}

type CachedDialer struct {
	*grpc.ClientConn
}

func (t CachedDialer) DialContext(ctx context.Context, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	return t.ClientConn, nil
}
