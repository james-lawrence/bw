package agent

import (
	"fmt"
	"net"
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
