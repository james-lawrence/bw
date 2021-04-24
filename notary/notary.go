package notary

import (
	"context"
	"crypto/x509/pkix"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/willf/bloom"
)

func AgentGrant(pub []byte) *Grant {
	return Grant{
		Permission:    agent(),
		Authorization: pub,
	}.EnsureDefaults()
}

// UserFull all the permissions.
func UserFull() *Permission {
	return &Permission{
		Grant:    true,
		Revoke:   true,
		Search:   true,
		Refresh:  true,
		Deploy:   true,
		Sync:     true,
		Autocert: false,
	}
}

func agent() *Permission {
	return &Permission{
		Grant:    false,
		Revoke:   false,
		Search:   false,
		Refresh:  false,
		Deploy:   true,
		Autocert: true,
		Sync:     true,
	}
}

// grant no permissions
func none() *Permission {
	return &Permission{}
}

type Bloomy interface {
	Test([]byte) bool
	Add([]byte) *bloom.BloomFilter
}

type SyncStorage interface {
	Sync(ctx context.Context, b Bloomy, c chan *Grant) (err error)
}

type storage interface {
	Lookup(fingerprint string) (*Grant, error)
	Insert(*Grant) (*Grant, error)
	Delete(*Grant) (*Grant, error)
}

type option func(*Service)

type authority interface {
	Create(duration time.Duration, bits int, options ...tlsx.X509Option) (ca, key, cert []byte, err error)
}

// New notary service.
func New(servername string, a authority, s storage, options ...option) Service {
	return Service{
		servername: servername,
		authority:  a,
		storage:    s,
		auth:       newAuth(s),
	}
}

// Service of a notary service
type Service struct {
	UnimplementedNotaryServer
	servername string
	authority  authority
	storage    storage
	auth       Auth
}

func (t Service) merge(options ...option) Service {
	for _, opt := range options {
		opt(&t)
	}

	return t
}

// Bind the service to the given grpc server.
func (t Service) Bind(s *grpc.Server, options ...option) {
	RegisterNotaryServer(s, t)
}

// Grant add a grant to the notary service.
func (t Service) Grant(ctx context.Context, req *GrantRequest) (_ *GrantResponse, err error) {
	var (
		g    *Grant
		resp GrantResponse
	)

	if p := t.auth.Authorize(ctx); !p.Grant {
		return &resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if req.Grant == nil {
		return &resp, errorsx.String("a grant must be provided")
	}

	if g, err = t.storage.Insert(req.Grant); err != nil {
		return &resp, err
	}

	return &GrantResponse{
		Grant: g,
	}, nil
}

// Revoke a grant from the notary service.
func (t Service) Revoke(ctx context.Context, req *RevokeRequest) (_ *RevokeResponse, err error) {
	var (
		g    *Grant
		resp RevokeResponse
	)

	if p := t.auth.Authorize(ctx); !p.Revoke {
		return &resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if g, err = t.storage.Delete(&Grant{Fingerprint: req.Fingerprint}); err != nil {
		return &resp, err
	}

	return &RevokeResponse{
		Grant: g,
	}, nil
}

// Refresh generate new TLS credentials
func (t Service) Refresh(ctx context.Context, req *RefreshRequest) (_ *RefreshResponse, err error) {
	var (
		resp RefreshResponse
	)

	log.Println("Notary.Refresh initated")
	defer log.Println("Notary.Refresh completed")
	if p := t.auth.Authorize(ctx); !p.Refresh {
		return &resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	caoptions := []tlsx.X509Option{
		tlsx.X509OptionSubject(pkix.Name{
			CommonName: t.servername,
		}),
	}

	if resp.Authority, resp.PrivateKey, resp.Certificate, err = t.authority.Create(20*time.Hour, 4096, caoptions...); err != nil {
		log.Println("failed to generate credentials", err)
		return nil, status.Error(codes.Unavailable, "authority not available")
	}

	return &resp, nil
}

// Search the notary service for grants.
func (t Service) Search(req *SearchRequest, dst Notary_SearchServer) (err error) {
	if p := t.auth.Authorize(dst.Context()); !p.Search {
		return status.Error(codes.PermissionDenied, "invalid credentials")
	}

	return status.Error(codes.Unimplemented, "not implemented")
}
