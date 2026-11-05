package testingx

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/akutz/memconn"
	"github.com/gofrs/uuid/v5"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const rootDir = ".tests"

// TempDir generates a tmp directory within the root testing directory for use in tests.
// the directory is removed when the test completes.
func TempDir(t testing.TB) (dir string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(rootDir, 0755))

	dir, err := os.MkdirTemp(rootDir, "tmp")
	require.NoError(t, err)

	t.Cleanup(func() {
		removeAll(dir)
	})

	return dir
}

// removeAll specs frequently leave background work behind,
// e.g.) a deploy that is still writing its files, which makes the removal race with it ("directory not empty").
// retry briefly, the writers finish quickly once the specs are done.
func removeAll(path string) error {
	var err error
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if err = os.RemoveAll(path); err == nil {
			return nil
		}
	}
	return err
}

// Cleanup used to cleanup the test directory after a package's tests are run. for use in TestMain.
func Cleanup() {
	removeAll(rootDir)
}

// RequireFilePermissions asserts the file at path has the given permissions.
func RequireFilePermissions(t testing.TB, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, expected, info.Mode().Perm())
}

// NewGRPCServer sets up a server and a connection.
func NewGRPCServer(t testing.TB, bind func(s *grpc.Server), options ...grpc.DialOption) (c *grpc.ClientConn, s *grpc.Server) {
	inmemconn := uuid.Must(uuid.NewV4()).String()
	l, err := memconn.Listen("memu", inmemconn)
	require.NoError(t, err)
	s = grpc.NewServer()
	bind(s)
	go func() {
		s.Serve(l)
	}()

	options = append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, address string) (conn net.Conn, err error) {
			ctx, done := context.WithTimeout(ctx, time.Second)
			defer done()
			return memconn.DialContext(ctx, "memu", address)
		}),
	}, options...)
	c, err = grpc.Dial(l.Addr().String(), options...)

	require.NoError(t, err)

	return c, s
}

// NewGRPCServer sets up a server and a dialer.
func NewGRPCServer2(t testing.TB, bind func(s *grpc.Server), options ...grpc.DialOption) (d dialers.Direct, s *grpc.Server) {
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	s = grpc.NewServer()
	bind(s)
	go func() {
		s.Serve(l)
	}()

	options = append(
		options,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDialer(func(addr string, to time.Duration) (c net.Conn, err error) {
			ctx, done := context.WithTimeout(context.Background(), to)
			defer done()
			return (&net.Dialer{}).DialContext(ctx, l.Addr().Network(), l.Addr().String())
		}),
	)
	return dialers.NewDirect(l.Addr().String(), dialers.NewDefaults(options...).Defaults()...), s
}

// GRPCCleanup shuts down the client and server. for use with defer.
func GRPCCleanup(t testing.TB, c *grpc.ClientConn, s *grpc.Server) {
	if c != nil {
		require.NoError(t, c.Close())
	}
	s.Stop()
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
