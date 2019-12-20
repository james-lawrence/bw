package daemons

import (
	"net"
	"path/filepath"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/notary"
)

// Discovery initiates the discovery backend.
func Discovery(ctx Context) (err error) {
	var (
		ns     notary.Storage
		bind   net.Listener
		creds  credentials.TransportCredentials
		server *grpc.Server
	)

	keepalive := grpc.KeepaliveParams(ctx.RPCKeepalive)

	if ns, err = notary.NewFromFile(filepath.Join(ctx.Config.Root, bw.DirAuthorizations), ctx.ConfigurationFile); err != nil {
		return err
	}

	if creds, err = GRPCGenServer(ctx.Config, tlsx.OptionVerifyClientIfGiven); err != nil {
		return err
	}

	if bind, err = net.Listen(ctx.Config.DiscoveryBind.Network(), ctx.Config.DiscoveryBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind discovery to %s", ctx.Config.DiscoveryBind)
	}

	server = grpc.NewServer(grpc.Creds(creds), keepalive)

	notary.New(
		ctx.Config.ServerName,
		certificatecache.NewAuthorityCache(ctx.Config.CredentialsDir),
		ns,
	).Bind(server)
	discovery.New(ctx.Cluster).Bind(server)

	// used to validate client certificates.
	discovery.NewAuthority(
		certificatecache.CAKeyPath(ctx.Config.CredentialsDir, certificatecache.DefaultTLSGeneratedCAProto),
	).Bind(server)

	ctx.grpc("discovery", server, bind)

	return nil
}
