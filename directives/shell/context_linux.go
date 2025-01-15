package shell

import (
	"net"
	"strings"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
)

func fqdn(hostname string) (string, error) {
	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return "", errors.Wrap(err, "failed to lookup ip for fqdn")
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				errorsx.MaybeLog(errors.Wrapf(err, "failed to marshal ip for fqdn: %s", ipv4.String()))
				continue
			}

			hosts, err := net.LookupAddr(string(ip))
			if err != nil {
				errorsx.MaybeLog(errors.Wrapf(err, "failed to lookup hosts for addr: %s", ipv4.String()))
				continue
			}

			for _, fqdn := range hosts {
				return strings.TrimSuffix(fqdn, "."), nil // return fqdn without trailing dot
			}
		}
	}

	// no FQDN found
	return "", nil
}
