package notary

import (
	"context"
	"log"

	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/internal/x/errorsx"
)

type dialer interface {
	DialContext(context.Context, ...grpc.DialOption) (*grpc.ClientConn, error)
}

// NewClient consumes a dialer
func NewClient(d dialer) Client {
	return Client{
		c: newCached(d),
	}
}

// Client for interacting with the notary service.
type Client struct {
	c cached
}

func (t Client) cached() (c NotaryClient, err error) {
	return maybeNotaryClient(t.c.cached())
}

// Grant the given key access to the system.
func (t Client) Grant(g *Grant) (_ *Grant, err error) {
	var (
		resp *GrantResponse
		c    NotaryClient
	)

	if c, err = t.cached(); err != nil {
		return nil, err
	}

	if resp, err = c.Grant(context.Background(), &GrantRequest{Grant: g}); err != nil {
		return nil, err
	}

	if resp.Grant == nil {
		return nil, errorsx.String("invalid response")
	}

	return resp.Grant, err
}

// Revoke the given key from the system.
func (t Client) Revoke(fingerprint string) (g *Grant, err error) {
	var (
		resp *RevokeResponse
		c    NotaryClient
	)

	if c, err = t.cached(); err != nil {
		return g, err
	}

	if resp, err = c.Revoke(context.Background(), &RevokeRequest{Fingerprint: fingerprint}); err != nil {
		return g, err
	}

	if resp.Grant == nil {
		return g, errorsx.String("invalid response")
	}

	return resp.Grant, err
}

// Refresh refresh TLS credentials.
func (t Client) Refresh() (ca, key, cert []byte, err error) {
	var (
		resp *RefreshResponse
		c    NotaryClient
	)

	log.Println("refresh initiated")
	defer log.Println("refresh completed")

	if c, err = t.cached(); err != nil {
		return ca, key, cert, err
	}

	if resp, err = c.Refresh(context.Background(), &RefreshRequest{}); err != nil {
		return ca, key, cert, err
	}

	return resp.Authority, resp.PrivateKey, resp.Certificate, err
}

// Search the service for a given key.
func (t Client) Search(req *SearchRequest) (resp *SearchResponse, err error) {
	return &SearchResponse{}, errorsx.String("not implemented")
}
