package daemons

import (
	"net"
	"path/filepath"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/notary"
)

// Discovery initiates the discovery backend.
func Discovery(ctx Context, c agent.Config, config string) (err error) {
	var (
		ns     notary.Storage
		bind   net.Listener
		creds  credentials.TransportCredentials
		server *grpc.Server
	)

	keepalive := grpc.KeepaliveParams(ctx.RPCKeepalive)

	if ns, err = notary.NewFromFile(filepath.Join(c.Root, bw.DirAuthorizations), config); err != nil {
		return err
	}

	if creds, err = GRPCGenServer(c, tlsx.OptionVerifyClientIfGiven); err != nil {
		return err
	}

	if bind, err = net.Listen(c.DiscoveryBind.Network(), c.DiscoveryBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind discovery to %s", c.DiscoveryBind)
	}

	server = grpc.NewServer(grpc.Creds(creds), keepalive)

	notary.New(
		c.ServerName,
		certificatecache.NewAuthorityCache(c.CredentialsDir),
		ns,
	).Bind(server)
	discovery.New(ctx.Cluster).Bind(server)

	// used to validate client certificates.
	discovery.NewAuthority(
		certificatecache.CAKeyPath(c.CredentialsDir, certificatecache.DefaultTLSGeneratedCAProto),
	).Bind(server)

	ctx.grpc("discovery", server, bind)

	return nil
}
