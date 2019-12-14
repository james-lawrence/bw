// Package discovery is used to provide information
// about the system to anyone.
package discovery

import (
	"context"
	"errors"

	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/systemx"
)

// cluster interface for the package.
type cluster interface {
	Quorum() []agent.Peer
}

type authorization interface {
	Authorized(context.Context) error
}

func peerToNode(p agent.Peer) Node {
	return Node{
		Ip:            p.Ip,
		Name:          p.Name,
		RPCPort:       p.RPCPort,
		RaftPort:      p.RaftPort,
		SWIMPort:      p.SWIMPort,
		TorrentPort:   p.TorrentPort,
		DiscoveryPort: p.DiscoveryPort,
	}
}

// nodeToPeer ...
func nodeToPeer(n Node) agent.Peer {
	return agent.Peer{
		Ip:            n.Ip,
		Name:          n.Name,
		RPCPort:       n.RPCPort,
		RaftPort:      n.RaftPort,
		SWIMPort:      n.SWIMPort,
		TorrentPort:   n.TorrentPort,
		DiscoveryPort: n.DiscoveryPort,
	}
}

// CheckCredentials against discovery
func CheckCredentials(address string, path string, options ...grpc.DialOption) (err error) {
	var (
		cc *grpc.ClientConn
		d  QuorumDialer
	)

	if !systemx.FileExists(path) {
		return nil
	}

	fingerprint := systemx.FileMD5(path)
	if fingerprint == "" {
		return errors.New("failed to generate fingerprint")
	}

	if d, err = NewQuorumDialer(address); err != nil {
		return err
	}
	defer d.Close()

	if cc, err = d.Dial(options...); err != nil {
		return err
	}
	defer cc.Close()

	_, err = NewAuthorityClient(cc).Check(context.Background(), &CheckRequest{Fingerprint: fingerprint})
	return err
}
