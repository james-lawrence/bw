package notary

import (
	"context"
	"crypto/tls"
	"log"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/james-lawrence/bw/internal/x/errorsx"
)

// DialOption ...
type DialOption func(*Dialer)

// DialOptionTLS ...
func DialOptionTLS(c *tls.Config) DialOption {
	return func(d *Dialer) {
		d.tls = grpc.WithTransportCredentials(credentials.NewTLS(c))
	}
}

// DialOptionCredentials credentials for the resulting client.
func DialOptionCredentials(c credentials.PerRPCCredentials) DialOption {
	return func(d *Dialer) {
		d.creds = grpc.WithPerRPCCredentials(c)
	}
}

// NewDialer used to establish connections to the cluster
// prior to TLS authentication of the client.
func NewDialer(proxy dialer, options ...DialOption) Dialer {
	d := Dialer{
		proxy: proxy,
		tls:   grpc.EmptyDialOption{},
		creds: grpc.EmptyDialOption{},
	}

	for _, opt := range options {
		opt(&d)
	}

	return d
}

// Dialer ...
type Dialer struct {
	proxy dialer
	creds grpc.DialOption
	tls   grpc.DialOption
}

// Dial ...
func (t Dialer) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	return t.proxy.Dial(append([]grpc.DialOption{t.tls, t.creds}, options...)...)
}

type dialer interface {
	Dial(...grpc.DialOption) (*grpc.ClientConn, error)
}

// NewClient consumes a dialer
func NewClient(d dialer) Client {
	return Client{
		dialer: d,
		m:      &sync.RWMutex{},
	}
}

// Client for interacting with the notary service.
type Client struct {
	dialer
	conn *grpc.ClientConn
	m    *sync.RWMutex
}

func (t Client) cached() (_ NotaryClient, err error) {
	t.m.RLock()
	c := t.conn
	t.m.RUnlock()

	if c != nil {
		return NewNotaryClient(c), nil
	}

	t.m.Lock()
	defer t.m.Unlock()

	if t.conn, err = t.dialer.Dial(); err != nil {
		return nil, err
	}

	return NewNotaryClient(t.conn), nil
}

// Grant the given key access to the system.
func (t Client) Grant(g Grant) (_ Grant, err error) {
	var (
		resp *GrantResponse
		c    NotaryClient
	)

	if c, err = t.cached(); err != nil {
		return g, err
	}

	if resp, err = c.Grant(context.Background(), &GrantRequest{Grant: &g}); err != nil {
		return g, err
	}

	if resp.Grant == nil {
		return g, errorsx.String("invalid response")
	}

	return *resp.Grant, err
}

// Revoke the given key from the system.
func (t Client) Revoke(fingerprint string) (g Grant, err error) {
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

	return *resp.Grant, err
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
func (t Client) Search(req SearchRequest) (resp SearchResponse, err error) {
	return resp, errorsx.String("not implemented")
}
