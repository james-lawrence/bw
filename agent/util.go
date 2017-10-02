package agent

import (
	"fmt"
	"net"

	"bitbucket.org/jatone/bearded-wookie/x/systemx"
)

// RPCAddress for peer.
func RPCAddress(p Peer) string {
	return net.JoinHostPort(p.Ip, fmt.Sprint(p.RPCPort))
}

// SWIMAddress for peer.
func SWIMAddress(p Peer) string {
	return net.JoinHostPort(p.Ip, fmt.Sprint(p.SWIMPort))
}

// RaftAddress for peer.
func RaftAddress(p Peer) string {
	return net.JoinHostPort(p.Ip, fmt.Sprint(p.RaftPort))
}

// LocalPeer build local peer.
func LocalPeer(id string) Peer {
	return Peer{
		Name:     id,
		Ip:       systemx.HostIP(systemx.HostnameOrLocalhost()).String(),
		RPCPort:  2000,
		SWIMPort: 2001,
		RaftPort: 2002,
		Status:   Peer_Ready,
	}
}
