package grpcx

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc/credentials"
)

// NewDebugTransportCredentials create debug credentials
func NewDebugTransportCredentials(creds credentials.TransportCredentials) credentials.TransportCredentials {
	return debugTransportCredentials{
		TransportCredentials: creds,
	}
}

type debugTransportCredentials struct {
	credentials.TransportCredentials
}

func (t debugTransportCredentials) ClientHandshake(ctx context.Context, n string, conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	log.Output(1, fmt.Sprintf("TransportCredentials.ClientHandshake initiated: %s", n))
	c, a, e := t.TransportCredentials.ClientHandshake(ctx, n, conn)
	defer log.Output(1, fmt.Sprintf("TransportCredentials.ClientHandshake completed: %s - %s", n, e))
	return c, a, e
}

func (t debugTransportCredentials) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	log.Output(1, "TransportCredentials.ServerHandshake initiated")
	defer log.Output(1, "TransportCredentials.ServerHandshake completed")
	return t.TransportCredentials.ServerHandshake(conn)
}

func (t debugTransportCredentials) Info() credentials.ProtocolInfo {
	log.Output(1, "TransportCredentials.Info initiated")
	defer log.Output(1, "TransportCredentials.Info completed")
	return t.TransportCredentials.Info()
}

func (t debugTransportCredentials) Clone() credentials.TransportCredentials {
	log.Output(1, "TransportCredentials.Clone initiated")
	defer log.Output(1, "TransportCredentials.Clone completed")
	return debugTransportCredentials{
		TransportCredentials: t.TransportCredentials.Clone(),
	}
}

func (t debugTransportCredentials) OverrideServerName(n string) error {
	log.Output(1, fmt.Sprintf("TransportCredentials.OverrideServerName initiated: %s", n))
	defer log.Output(1, fmt.Sprintf("TransportCredentials.OverrideServerName completed: %s", n))
	return t.TransportCredentials.OverrideServerName(n)
}
