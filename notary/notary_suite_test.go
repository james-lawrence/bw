package notary_test

import (
	"context"
	"io"
	"log"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/notary"
	"google.golang.org/grpc"
)

func TestNotary(t *testing.T) {
	log.SetOutput(io.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notary Suite")
}

var _ = SynchronizedAfterSuite(func() {}, testingx.Cleanup)

// QuickService ...
func QuickService() (notary.Client, func()) {
	pkey, storage := QuickStorage()

	ss, err := notary.NewSigner(pkey)
	Expect(err).To(Succeed())

	svc := notary.New("", nil, storage)
	conn, server := testingx.NewGRPCServer(func(s *grpc.Server) {
		svc.Bind(s)
	}, grpc.WithPerRPCCredentials(ss))

	return notary.NewClient(NewStaticDialer(conn)), func() { testingx.GRPCCleanup(conn, server) }
}

type storage interface {
	Lookup(fingerprint string) (*notary.Grant, error)
	Insert(*notary.Grant) (*notary.Grant, error)
	Delete(*notary.Grant) (*notary.Grant, error)
}

// QuickStorage ...
func QuickStorage() (encoded []byte, s notary.Directory) {
	pkey, err := rsax.Generate(1024)
	Expect(err).To(Succeed())
	pubkey, err := sshx.PublicKey(pkey)
	Expect(err).To(Succeed())

	storage := notary.NewDirectory(testingx.TempDir())
	_, err = storage.Insert(&notary.Grant{Authorization: pubkey, Permission: notary.UserFull()})
	Expect(err).To(Succeed())

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

func QuickSigner() (notary.Signer, error) {
	pkey, err := rsax.UnsafeAuto()
	Expect(err).To(Succeed())
	return notary.NewSigner(pkey)
}

// QuickKey generate a quick keypair.
func QuickKey() ([]byte, []byte) {
	// generate a new key to add.
	pkey, err := rsax.UnsafeAuto()
	Expect(err).To(Succeed())
	pubkey, err := sshx.PublicKey(pkey)
	Expect(err).To(Succeed())

	return pkey, pubkey
}

// QuickGrant generate a grant
func QuickGrant() *notary.Grant {
	_, pubkey := QuickKey()
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
