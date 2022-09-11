package shell

import (
	"log"
	"net"
	"strings"

	"github.com/james-lawrence/bw/internal/logx"
	"github.com/pkg/errors"
)

// macosx tends to not properly set the hostname.
func fqdn(hostname string) (string, error) {
	addrs, err := net.LookupIP(hostname)
	if err != nil {
		log.Println(errors.Wrap(err, "failed to lookup ip for fqdn, defaulting to localhost"))
		return "", nil
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				logx.MaybeLog(errors.Wrapf(err, "failed to marshal ip for fqdn: %s", ipv4.String()))
				continue
			}

			hosts, err := net.LookupAddr(string(ip))
			if err != nil {
				logx.MaybeLog(errors.Wrapf(err, "failed to lookup hosts for addr: %s", ipv4.String()))
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

func machineID() string {
	log.Println("machine id not supported on darwin systems")
	return ""
}
