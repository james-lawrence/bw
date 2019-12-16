package notary_test

import (
	"io/ioutil"
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw/internal/x/sshx"
	"github.com/james-lawrence/bw/internal/x/testingx"
	"github.com/james-lawrence/bw/notary"
	"google.golang.org/grpc"
)

func TestNotary(t *testing.T) {
	log.SetOutput(ioutil.Discard)
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

// QuickStorage ...
func QuickStorage() (encoded []byte, s notary.Storage) {
	pkey, err := sshx.Generate(1024)
	Expect(err).To(Succeed())
	pubkey, err := sshx.PublicKey(pkey)
	Expect(err).To(Succeed())

	storage := notary.NewStorage(testingx.TempDir())
	_, err = storage.Insert(notary.Grant{Authorization: pubkey, Permission: notary.PermAll()})
	Expect(err).To(Succeed())

	return pkey, storage
}

// QuickKey generate a quick keypair.
func QuickKey() ([]byte, []byte) {
	// generate a new key to add.
	pkey, err := sshx.UnsafeAuto()
	Expect(err).To(Succeed())
	pubkey, err := sshx.PublicKey(pkey)
	Expect(err).To(Succeed())

	return pkey, pubkey
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
