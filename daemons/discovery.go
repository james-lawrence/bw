package daemons

import (
	"net"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// Discovery initiates the discovery backend.
func Discovery(ctx Context, config string) (err error) {
	var (
		c      agent.Config
		ns     notary.Storage
		bind   net.Listener
		creds  credentials.TransportCredentials
		server *grpc.Server
	)

	keepalive := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: 1 * time.Hour,
		Time:              1 * time.Minute,
		Timeout:           2 * time.Minute,
	})

	if err = bw.ExpandAndDecodeFile(config, &c); err != nil {
		return err
	}

	if ns, err = notary.NewFromFile(config); err != nil {
		return err
	}

	if creds, err = GRPCGenServer(c); err != nil {
		return err
	}

	if bind, err = net.Listen(c.DiscoveryBind.Network(), c.DiscoveryBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind discovery to %s", c.DiscoveryBind)
	}

	server = grpc.NewServer(grpc.Creds(creds), keepalive)
	notary.New(ns).Bind(server)

	ctx.grpc("discovery", server, bind)

	return nil
}
