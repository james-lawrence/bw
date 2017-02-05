package upnp

import (
	"fmt"
	"net"

	"github.com/huin/goupnp/dcps/internetgateway2"
	"github.com/pkg/errors"
)

const (
	tcp = "TCP"
	udp = "UDP"
)

// AddTCP ...
func AddTCP(internal *net.TCPAddr) (*net.TCPAddr, error) {
	var (
		err     error
		clients []*internetgateway2.WANIPConnection1
	)

	if clients, err = newClients(); err != nil {
		return nil, err
	}

	ip, port, err := addPortMapping(clients[0], tcp, uint16(internal.Port), internal.IP)
	if err != nil {
		return nil, err
	}

	return &net.TCPAddr{
		Port: int(port),
		IP:   ip,
	}, nil
}

// DeleteTCP ...
func DeleteTCP(external *net.TCPAddr) error {
	var (
		err     error
		clients []*internetgateway2.WANIPConnection1
	)

	if external == nil {
		return nil
	}

	if clients, err = newClients(); err != nil {
		return err
	}

	return deletePortMapping(clients[0], tcp, uint16(external.Port))
}

// AddUDP ...
func AddUDP(internal *net.UDPAddr) (*net.UDPAddr, error) {
	var (
		err     error
		clients []*internetgateway2.WANIPConnection1
	)

	if clients, err = newClients(); err != nil {
		return nil, err
	}

	ip, port, err := addPortMapping(clients[0], udp, uint16(internal.Port), internal.IP)
	if err != nil {
		return nil, err
	}

	return &net.UDPAddr{
		Port: int(port),
		IP:   ip,
	}, nil
}

// DeleteUDP ...
func DeleteUDP(external *net.UDPAddr) error {
	var (
		err     error
		clients []*internetgateway2.WANIPConnection1
	)

	if external == nil {
		return nil
	}

	if clients, err = newClients(); err != nil {
		return err
	}

	return deletePortMapping(clients[0], udp, uint16(external.Port))
}

func newClients() ([]*internetgateway2.WANIPConnection1, error) {
	var (
		err     error
		errs    []error
		clients []*internetgateway2.WANIPConnection1
	)

	if clients, errs, err = internetgateway2.NewWANIPConnection1Clients(); err != nil {
		return clients, err
	}

	if len(errs) > 0 {
		return clients, errors.Wrapf(collapseErrors(errs...), "upnp providers detected but failed to connect")
	}

	if len(clients) == 0 {
		return clients, errors.New("no upnp providers detected")
	}

	return clients, nil
}

func deletePortMapping(client *internetgateway2.WANIPConnection1, protocol string, port uint16) error {
	return client.DeletePortMapping("", port, protocol)
}

func addPortMapping(client *internetgateway2.WANIPConnection1, protocol string, internalPort uint16, internalIP net.IP) (net.IP, uint16, error) {
	var (
		err        error
		reserved   uint16
		externalIP string
	)

	if externalIP, err = client.GetExternalIPAddress(); err != nil {
		return nil, reserved, errors.Wrap(err, "failed to get externalIP")
	}

	err = client.AddPortMapping(
		"",
		internalPort,
		protocol,
		internalPort,
		internalIP.String(),
		true,
		"bearded-wookie",
		0,
	)

	return net.ParseIP(externalIP), internalPort, err
}

func collapseErrors(errs ...error) error {
	var (
		err error
		s   string
	)
	for _, err = range errs {
		s += fmt.Sprintf("%s\n", err)
	}
	return errors.New(s)
}
