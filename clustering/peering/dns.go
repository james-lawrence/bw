package peering

import (
	"context"
	"net"
	"strconv"

	"github.com/pkg/errors"
)

// NewDNS create a new DNS peering strategy
func NewDNS(p int, hosts ...string) DNS {
	return DNS{
		Port:  p,
		Hosts: hosts,
	}
}

// DNS based peering
type DNS struct {
	Port  int // port to connect to.
	Hosts []string
}

// Peers - reads peers from a dns record.
func (t DNS) Peers(context.Context) (results []string, err error) {
	var (
		ips []net.IP
	)

	for _, host := range t.Hosts {
		if ips, err = net.LookupIP(host); err != nil {
			return results, errors.WithStack(err)
		}

		ps := strconv.Itoa(t.Port)
		for _, ip := range ips {
			results = append(results, net.JoinHostPort(ip.String(), ps))
		}
	}

	return results, nil
}
