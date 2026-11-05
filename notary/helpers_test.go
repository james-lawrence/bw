package notary_test

import (
	"context"
	"io"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/notary"
	"google.golang.org/grpc"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	code := m.Run()
	testingx.Cleanup()
	os.Exit(code)
}

// QuickService ...
func QuickService(t testing.TB) (notary.Client, func()) {
	pkey, storage := QuickStorage(t)

	ss, err := notary.NewSigner(pkey)
	require.NoError(t, err)

	svc := notary.New("", nil, storage)
	conn, server := testingx.NewGRPCServer(t, func(s *grpc.Server) {
		svc.Bind(s)
	}, grpc.WithPerRPCCredentials(ss))

	return notary.NewClient(NewStaticDialer(conn)), func() { testingx.GRPCCleanup(t, conn, server) }
}

type storage interface {
	Lookup(fingerprint string) (*notary.Grant, error)
	Insert(*notary.Grant) (*notary.Grant, error)
	Delete(*notary.Grant) (*notary.Grant, error)
	Sync(ctx context.Context, b notary.Bloomy, c chan *notary.Grant) (err error)
}

// QuickStorage ...
func QuickStorage(t testing.TB) (encoded []byte, s notary.Directory) {
	pkey, err := rsax.Generate(1024)
	require.NoError(t, err)
	pubkey, err := sshx.PublicKey(pkey)
	require.NoError(t, err)

	storage := notary.NewDirectory(testingx.TempDir(t))
	_, err = storage.Insert(&notary.Grant{Authorization: pubkey, Permission: notary.UserFull()})
	require.NoError(t, err)

	return pkey, storage
}

type staticauth struct {
	*notary.Permission
}

func (t staticauth) Authorize(ctx context.Context) *notary.Permission {
	return t.Permission
}

// grant all permissions
func all() *notary.Permission {
	return &notary.Permission{
		Grant:    true,
		Revoke:   true,
		Search:   true,
		Refresh:  true,
		Deploy:   true,
		Autocert: true,
		Sync:     true,
	}
}

func QuickSigner(t testing.TB) (notary.Signer, error) {
	pkey, err := rsax.UnsafeAuto()
	require.NoError(t, err)
	return notary.NewSigner(pkey)
}

// QuickKey generate a quick keypair.
func QuickKey(t testing.TB) ([]byte, []byte) {
	// generate a new key to add.
	pkey, err := rsax.UnsafeAuto()
	require.NoError(t, err)
	pubkey, err := sshx.PublicKey(pkey)
	require.NoError(t, err)

	return pkey, pubkey
}

// QuickGrant generate a grant
func QuickGrant(t testing.TB) *notary.Grant {
	_, pubkey := QuickKey(t)
	return (&notary.Grant{
		Permission:    all(),
		Authorization: pubkey,
	}).EnsureDefaults()
}

// NewStaticDialer ...
func NewStaticDialer(c *grpc.ClientConn) StaticDialer {
	return StaticDialer{conn: c}
}

// StaticDialer
type StaticDialer struct {
	conn *grpc.ClientConn
}

func (t StaticDialer) Dial(_ ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	return t.conn, nil
}

func (t StaticDialer) DialContext(_ context.Context, _ ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	return t.conn, nil
}
