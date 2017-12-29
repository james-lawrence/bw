// Package dns manages a DNS entry for the cluster.
// keeping it up to date with a random sampling of nodes.
package dns

import (
	"net"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/miekg/dns"
)

type cluster interface {
	GetN(n int, key []byte) []*memberlist.Node
}

// Option set common options for DNS managers.
type Option func(*config)

// OptionMaximumNodes number of nodes to store in the dns entry.
func OptionMaximumNodes(n int) Option {
	return func(c *config) {
		c.MaximumNodes = n
	}
}

// OptionFQDN name to use.
func OptionFQDN(name string) Option {
	return func(c *config) {
		c.FQDN = name
	}
}

// OptionTTL ttl for the record
func OptionTTL(ttl uint32) Option {
	return func(c *config) {
		c.TTL = ttl
	}
}

type config struct {
	MaximumNodes int
	FQDN         string
	TTL          uint32
}

func (t config) merge(options ...Option) config {
	for _, opt := range options {
		opt(&t)
	}

	return t
}

func (t config) peersToBind(peers ...agent.Peer) []dns.A {
	rrset := make([]dns.A, 0, len(peers))
	for _, peer := range peers {
		rrset = append(rrset, dns.A{
			Hdr: dns.RR_Header{
				Name:   t.FQDN,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    t.TTL,
			},
			A: net.ParseIP(peer.Ip),
		})
	}

	return rrset
}

// Sampler ...
type Sampler interface {
	Sample(c cluster) error
}

type noopSampler struct{}

func (t noopSampler) Sample(c cluster) error {
	return nil
}

// MaybeSample ...
func MaybeSample(s Sampler, err error) func(cluster) error {
	return func(c cluster) error {
		if err == nil {
			s.Sample(c)
		}

		return err
	}
}
