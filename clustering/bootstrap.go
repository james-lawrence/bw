package clustering

import (
	"fmt"
	"log"
	"net"
	"strconv"
)

// ErrPeeringOptionsExhausted returned by bootstrap methods when the strategies for peering have been exhausted.
var ErrPeeringOptionsExhausted = fmt.Errorf("ran out of peering options, unable to locate peers")

// BootstrapOption option for bootstrapping a clusters
type BootstrapOption func(*bootstrap)

// BootstrapOptionEnableSingleNode - allows single node operation.
func BootstrapOptionEnableSingleNode(t bool) BootstrapOption {
	return func(b *bootstrap) {
		b.SingleNode = t
	}
}

// BootstrapOptionPeeringStrategies - set the strategies for peering.
func BootstrapOptionPeeringStrategies(p ...peering) BootstrapOption {
	return func(b *bootstrap) {
		b.Peering = p
	}
}

type bootstrap struct {
	SingleNode bool
	Peering    []peering
}

// Bootstrap - bootstraps the provided cluster using the options provided.
func Bootstrap(c Cluster, options ...BootstrapOption) error {
	var (
		err    error
		joined int
		peers  []string
		b      bootstrap
	)

	for _, opt := range options {
		opt(&b)
	}

	for _, s := range b.Peering {
		if peers, err = s.Peers(); err != nil {
			log.Printf("failed to load peers: %T: %s\n", s, err)
			continue
		}

		log.Printf("%T: located %d peers\n", s, len(peers))
		if joined, err = c.list.Join(peers); err != nil {
			log.Printf("failed to join peers: %T: %s\n", s, err)
			continue
		}

		if joined == 0 {
			log.Printf("join succeeded but no peers were located: %T\n", s)
			continue
		}

		break
	}

	if b.SingleNode && joined == 0 {
		log.Println("unable to locate peers using the provided peering strategies, continuing in single node mode due to settings")
		return nil
	}

	if joined == 0 {
		return ErrPeeringOptionsExhausted
	}

	return nil
}

// Peers converts the peers into an array of host:port.
func Peers(c cluster) []string {
	peers := c.Members()
	list := make([]string, 0, len(peers))
	for _, peer := range peers {
		list = append(list, net.JoinHostPort(peer.Addr.String(), strconv.Itoa(int(peer.Port))))
	}
	return list
}
