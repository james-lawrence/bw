package shell

import (
	"log"
	"net"
	"os"
	"strings"

	"github.com/james-lawrence/bw/internal/x/logx"
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
	var (
		err error
		raw []byte
	)

	if raw, err = os.ReadFile("/etc/machine-id"); err != nil {
		log.Println("failed to read machine id, defaulting to empty string", err)
		return ""
	}

	return strings.TrimSpace(string(raw))
}
