package p2pping

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/libp2p/go-libp2p-core/crypto"
	cpb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/sec"
	pb "github.com/libp2p/go-libp2p-core/sec/insecure/pb"
	"github.com/libp2p/go-msgio"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	mss "github.com/multiformats/go-multistream"
	"github.com/pkg/errors"
)

// ID security protocol ID.
const ID = "/identify/1.0.0"

// Transport ping Security Protocol is used to bootstrap peers using standard network transports.
type Transport struct {
	id      peer.ID
	privkey crypto.PrivKey
}

// SecureInbound ...
func (t *Transport) SecureInbound(ctx context.Context, insecure net.Conn) (sec.SecureConn, error) {
	conn := &Conn{
		Conn:         insecure,
		local:        t.id,
		localPrivKey: t.privkey,
	}

	log.Println("ID Transport - inbound", insecure.RemoteAddr().String(), "->", insecure.LocalAddr().String())

	if err := conn.runHandshakeSync(); err != nil {
		return nil, err
	}
	defer conn.Close()

	return conn, errors.New("dropping identify connection")
}

// SecureOutbound used to identify a server without establishing a full connection.
func (t *Transport) SecureOutbound(ctx context.Context, insecure net.Conn, p peer.ID) (sec.SecureConn, error) {
	log.Println("ID Transport - outbound", insecure.LocalAddr().String(), "->", insecure.RemoteAddr().String())
	conn := &Conn{
		Conn:         insecure,
		local:        t.id,
		localPrivKey: t.privkey,
	}

	if err := conn.runHandshakeSync(); err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn, errors.New("dropping identify connection")
}

// NewIDSec creates a new Noise transport using the given private key as its
// libp2p identity key.
func NewIDSec(privkey crypto.PrivKey) (*Transport, error) {
	local, err := peer.IDFromPublicKey(privkey.GetPublic())
	return &Transport{
		id:      local,
		privkey: privkey,
	}, err
}

// Identify handshakes an address to determine id and public key.
func Identify(ctx context.Context, privkey crypto.PrivKey, addr net.Addr) (id peer.AddrInfo, err error) {
	var (
		local  peer.ID
		raddr  ma.Multiaddr
		dialer manet.Dialer
		conn   manet.Conn
	)

	if local, err = peer.IDFromPublicKey(privkey.GetPublic()); err != nil {
		return id, err
	}

	if raddr, err = manet.FromNetAddr(addr); err != nil {
		return id, errors.Wrap(err, "failed to create multi address")
	}

	if conn, err = dialer.DialContext(ctx, raddr); err != nil {
		return id, errors.Wrap(err, "failed to create multiaddress")
	}
	defer conn.Close()

	_, err = mss.SelectOneOf([]string{
		ID,
	}, conn)

	sconn := &Conn{
		Conn:         conn,
		local:        local,
		localPrivKey: privkey,
	}
	defer sconn.Close()

	if err := sconn.runHandshakeSync(); err != nil {
		return id, err
	}

	pid, err := peer.IDFromPublicKey(sconn.RemotePublicKey())

	return peer.AddrInfo{
		ID: pid,
		Addrs: []ma.Multiaddr{
			raddr,
		},
	}, nil
}

// Conn is the connection type returned by the insecure transport.
type Conn struct {
	net.Conn

	local  peer.ID
	remote peer.ID

	localPrivKey crypto.PrivKey
	remotePubKey crypto.PubKey
}

func makeExchangeMessage(id peer.ID, privkey crypto.PrivKey) (_ *pb.Exchange, err error) {
	var (
		encoded *cpb.PublicKey
	)

	// if we are an insecure connect we'll just return the id.
	if privkey == nil {
		return &pb.Exchange{
			Id: []byte(id),
		}, nil
	}

	pubkey := privkey.GetPublic()
	if encoded, err = crypto.PublicKeyToProto(pubkey); err != nil {
		return nil, err
	}

	if id, err = peer.IDFromPublicKey(pubkey); err != nil {
		return nil, err
	}

	return &pb.Exchange{
		Id:     []byte(id),
		Pubkey: encoded,
	}, nil
}

func (ic *Conn) runHandshakeSync() error {
	// Generate an Exchange message
	msg, err := makeExchangeMessage(ic.local, ic.localPrivKey)
	if err != nil {
		return err
	}

	// Send our Exchange and read theirs
	remoteMsg, err := readWriteMsg(ic.Conn, msg)
	if err != nil {
		return err
	}

	// Pull remote ID and public key from message
	remotePubkey, err := crypto.PublicKeyFromProto(remoteMsg.Pubkey)
	if err != nil {
		return err
	}

	remoteID, err := peer.IDFromBytes(remoteMsg.Id)
	if err != nil {
		return err
	}

	// Validate that ID matches public key
	if !remoteID.MatchesPublicKey(remotePubkey) {
		calculatedID, _ := peer.IDFromPublicKey(remotePubkey)
		return fmt.Errorf("remote peer id does not match public key. id=%s calculated_id=%s",
			remoteID, calculatedID)
	}

	// Add remote ID and key to conn state
	ic.remotePubKey = remotePubkey
	ic.remote = remoteID
	return nil
}

// read and write a message at the same time.
func readWriteMsg(rw io.ReadWriter, out *pb.Exchange) (*pb.Exchange, error) {
	const maxMessageSize = 1 << 16

	outBytes, err := out.Marshal()
	if err != nil {
		return nil, err
	}
	wresult := make(chan error)
	go func() {
		w := msgio.NewVarintWriter(rw)
		wresult <- w.WriteMsg(outBytes)
	}()

	r := msgio.NewVarintReaderSize(rw, maxMessageSize)
	msg, err1 := r.ReadMsg()

	// Always wait for the read to finish.
	err2 := <-wresult

	if err1 != nil {
		return nil, err1
	}
	if err2 != nil {
		r.ReleaseMsg(msg)
		return nil, err2
	}
	inMsg := new(pb.Exchange)
	err = inMsg.Unmarshal(msg)
	return inMsg, err
}

// LocalPeer returns the local peer ID.
func (ic *Conn) LocalPeer() peer.ID {
	return ic.local
}

// RemotePeer returns the remote peer ID if we initiated the dial. Otherwise, it
// returns "" (because this connection isn't actually secure).
func (ic *Conn) RemotePeer() peer.ID {
	return ic.remote
}

// RemotePublicKey returns whatever public key was given by the remote peer.
// Note that no verification of ownership is done, as this connection is not secure.
func (ic *Conn) RemotePublicKey() crypto.PubKey {
	return ic.remotePubKey
}

// LocalPrivateKey returns the private key for the local peer.
func (ic *Conn) LocalPrivateKey() crypto.PrivKey {
	return ic.localPrivKey
}
