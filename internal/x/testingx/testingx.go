package testingx

import (
	"io/ioutil"
	"net"
	"os"

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
	l, err := net.Listen("tcp", ":0")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	s = grpc.NewServer()
	bind(s)
	go func() {
		s.Serve(l)
	}()

	options = append([]grpc.DialOption{grpc.WithInsecure()}, options...)
	c, err = grpc.Dial(l.Addr().String(), options...)

	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return c, s
}

// GRPCCleanup shuts down the client and server. for use with defer.
func GRPCCleanup(c *grpc.ClientConn, s *grpc.Server) {
	gomega.Expect(c.Close()).ToNot(gomega.HaveOccurred())
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
