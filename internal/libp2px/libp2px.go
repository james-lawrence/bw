package libp2px

import (
	"net"

	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p/config"
	"github.com/pkg/errors"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

// The various protocols defined for bearded wookie
const (
	Discovery protocol.ID = "bw.discovery"
	Agent                 = "bw.agent"
	Autocert              = "bw.autocert"
)

// RSAIdentityPriv decodes an rsa private key and creates a libp2p crypto private key from it.
func RSAIdentityPriv(encoded []byte, err error) (crypto.PrivKey, error) {
	return crypto.UnmarshalRsaPrivateKey(rsax.DecodePKCS1PrivateKey(encoded))
}

// RSAIdentity decodes an rsa private key and creates a libp2p identity from it.
func RSAIdentity(encoded []byte, err error) libp2p.Option {
	return func(cfg *config.Config) error {
		var (
			priv crypto.PrivKey
		)

		if priv, err = crypto.UnmarshalRsaPrivateKey(rsax.DecodePKCS1PrivateKey(encoded)); err != nil {
			return errors.Wrap(err, "failed to decode private rsa key")
		}

		return libp2p.Identity(priv)(cfg)
	}
}

// ListenNetAddrs listen to a set of net.Addr.
func ListenNetAddrs(addrs ...net.Addr) libp2p.Option {
	return func(cfg *config.Config) (err error) {
		saddrs := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			var (
				m ma.Multiaddr
			)
			if m, err = manet.FromNetAddr(addr); err != nil {
				return err
			}
			saddrs = append(saddrs, m.String())
		}

		return libp2p.ListenAddrStrings(saddrs...)(cfg)
	}
}
