package daemons

import (
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/internal/x/tlsx"
)

// Autocert - blocking rpc endpoint until a valid certificate is generated.
func Autocert(ctx Context) (err error) {
	var (
		bind    net.Listener
		acmesvc acme.Service
	)

	if acmesvc, err = acme.ReadConfig(ctx.Config, ctx.ConfigurationFile); err != nil {
		return err
	}

	keepalive := grpc.KeepaliveParams(ctx.RPCKeepalive)

	creds, err := tlsx.Clone(ctx.RPCCredentials, tlsx.OptionVerifyClientIfGiven)
	if err != nil {
		return err
	}

	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(creds)), keepalive)
	acme.RegisterACMEServer(server, acmesvc)

	if bind, err = net.Listen(ctx.Config.AutocertBind.Network(), ctx.Config.AutocertBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", ctx.Config.AutocertBind)
	}

	ctx.grpc("autocert", server, bind)

	return nil
}
