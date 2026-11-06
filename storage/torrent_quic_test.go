package storage

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/quicx"
	"github.com/james-lawrence/bw/internal/torrentx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/torrent"
	"github.com/stretchr/testify/require"
)

func quicTLS(t *testing.T) (server, client *tls.Config) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "bw"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: []string{"bw"}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	require.NoError(t, err)
	crt, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AddCert(crt)

	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: k}}},
		&tls.Config{RootCAs: pool, ServerName: "bw"}
}

type transports struct {
	tcp  bool // listen for tcp.
	quic bool // listen for quic.
	// dial prefers quic when it is listening and falls back to tcp.
	dialquic bool
}

// newMuxedTorrent a real torrent client wired as the agent does (daemons.Torrent): the torrent protocol is bound on the
// muxer, which accepts tcp connections and quic streams on the same port, and dials through the muxer.
// the returned counter is the number of connections dialed over tcp.
func newMuxedTorrent(t *testing.T, tr transports, server, client *tls.Config, cls func(port uint32) staticCluster) (TorrentConfig, *agent.Peer, *atomic.Int64) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var (
		qe     *quicx.Endpoint
		pc     net.PacketConn
		tcp    net.Listener
		err    error
		listen []net.Listener
	)

	// quic and tcp share the port number.
	for attempt := 0; ; attempt++ {
		if tr.quic {
			pc, err = net.ListenPacket("udp", "127.0.0.1:0")
			require.NoError(t, err)
		}

		if !tr.tcp {
			break
		}

		addr := "127.0.0.1:0"
		if pc != nil {
			addr = pc.LocalAddr().String()
		}

		if tcp, err = net.Listen("tcp", addr); err == nil {
			break
		}

		require.Less(t, attempt, 20, "unable to find a port free for tcp and udp")
		if pc != nil {
			pc.Close()
		}
	}

	if tcp != nil {
		listen = append(listen, tcp)
	}

	if pc != nil {
		qe, err = quicx.New(pc, server, client)
		require.NoError(t, err)
		listen = append(listen, qe)
	}

	var addr net.Addr
	switch {
	case tcp != nil:
		addr = tcp.Addr()
	default:
		addr = qe.Addr()
	}

	tcpdials := new(atomic.Int64)
	var dial interface {
		DialContext(ctx context.Context, network string, address string) (net.Conn, error)
	} = countingDialer{n: tcpdials}
	if tr.dialquic {
		require.NotNil(t, qe)
		dial = qe.Fallback(dial)
	}

	m := muxer.New()
	bind, err := m.Bind(bw.ProtocolTorrent, addr)
	require.NoError(t, err)
	for _, l := range listen {
		go muxer.Listen(ctx, m, l)
	}

	c, me := newTorrentFromSocket(t, addr, torrentx.Socket{
		Listener: bind,
		Dialer:   muxer.NewDialer(bw.ProtocolTorrent, dial),
	}, cls)
	return c, me, tcpdials
}

type countingDialer struct{ n *atomic.Int64 }

func (t countingDialer) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	t.n.Add(1)
	return (&net.Dialer{}).DialContext(ctx, network, address)
}

func newTorrentFromSocket(t *testing.T, addr net.Addr, s torrentx.Socket, cls func(port uint32) staticCluster) (TorrentConfig, *agent.Peer) {
	var port int
	switch a := addr.(type) {
	case *net.TCPAddr:
		port = a.Port
	case *net.UDPAddr:
		port = a.Port
	default:
		t.Fatalf("unexpected address %T", addr)
	}

	me := &agent.Peer{Ip: "127.0.0.1", P2PPort: uint32(port)}
	c, err := NewTorrent(
		cls(me.P2PPort),
		TorrentOptionBind(torrent.NewSocketsBind(s)),
		TorrentOptionDataDir(filepath.Join(t.TempDir(), "torrents")),
	)
	require.NoError(t, err)
	t.Cleanup(func() { c.client.Close() })
	return c, me
}

// two real torrent clients transferring an archive, over every combination of transports the agent supports.
func TestTorrentTransferOverQUICAndTCP(t *testing.T) {
	for _, tc := range []struct {
		name             string
		seeder, receiver transports
		expectTCP        bool // the receiver is expected to have fallen back to tcp.
	}{
		{"quic", transports{quic: true, dialquic: true}, transports{quic: true, dialquic: true}, false},
		{"tcp", transports{tcp: true}, transports{tcp: true}, true},
		{"quic is preferred when both are available", transports{tcp: true, quic: true, dialquic: true}, transports{tcp: true, quic: true, dialquic: true}, false},
		{"fallback to tcp when the seeder has no quic", transports{tcp: true}, transports{tcp: true, quic: true, dialquic: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, client := quicTLS(t) // every node trusts the same authority.
			seeder, seederPeer, _ := newMuxedTorrent(t, tc.seeder, server, client, func(uint32) staticCluster { return staticCluster{} })

			payload := make([]byte, 8<<20)
			_, err := rand.Read(payload)
			require.NoError(t, err)
			up, err := seeder.Uploader().NewUpload(uint64(len(payload)))
			require.NoError(t, err)
			_, err = up.Upload(bytes.NewReader(payload))
			require.NoError(t, err)
			_, magnet, err := up.Info()
			require.NoError(t, err)

			receiver, _, tcpdials := newMuxedTorrent(t, tc.receiver, server, client, func(uint32) staticCluster {
				return staticCluster{peers: []*agent.Peer{seederPeer}}
			})

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			r := receiver.Downloader().New().Download(ctx, &agent.Archive{Location: magnet})
			defer r.Close()

			got, err := io.ReadAll(r)
			require.NoError(t, err)
			require.True(t, bytes.Equal(payload, got), "downloaded %d bytes, expected %d", len(got), len(payload))
			if tc.expectTCP {
				require.NotZero(t, tcpdials.Load(), "expected the transfer to use tcp")
			} else {
				require.Zero(t, tcpdials.Load(), "expected the transfer to stay on quic")
			}
		})
	}
}
