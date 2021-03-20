package notary

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// NewProxy new proxy service.
func NewProxy(d dialer) Proxy {
	return Proxy{
		c: newCached(
			d,
		),
	}
}

// Proxy service proxies the request to another service based on the dialer.
type Proxy struct {
	UnimplementedNotaryServer
	c cached
}

// Bind the service to the given grpc server.
func (t Proxy) Bind(s *grpc.Server) {
	RegisterNotaryServer(s, t)
}

func (t Proxy) cached() (c NotaryClient, err error) {
	return maybeNotaryClient(t.c.cached())
}

func (t Proxy) metadata(incoming context.Context) context.Context {
	if md, ok := metadata.FromIncomingContext(incoming); ok {
		return metadata.NewOutgoingContext(context.Background(), md)
	}

	return context.Background()
}

// Grant add a grant to the notary service.
func (t Proxy) Grant(ctx context.Context, req *GrantRequest) (resp *GrantResponse, err error) {
	var (
		c NotaryClient
	)
	if c, err = t.cached(); err != nil {
		return nil, err
	}
	return c.Grant(t.metadata(ctx), req)
}

// Revoke a grant from the notary service.
func (t Proxy) Revoke(ctx context.Context, req *RevokeRequest) (resp *RevokeResponse, err error) {
	var (
		c NotaryClient
	)
	if c, err = t.cached(); err != nil {
		return nil, err
	}
	return c.Revoke(t.metadata(ctx), req)
}

// Refresh generate new TLS credentials
func (t Proxy) Refresh(ctx context.Context, req *RefreshRequest) (resp *RefreshResponse, err error) {
	var (
		c NotaryClient
	)
	if c, err = t.cached(); err != nil {
		return nil, err
	}
	return c.Refresh(t.metadata(ctx), req)
}

// Search the notary service for grants.
func (t Proxy) Search(req *SearchRequest, dst Notary_SearchServer) (err error) {
	return status.Error(codes.Unimplemented, "not implemented")
}
