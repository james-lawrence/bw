package notary

import (
	"context"
	"crypto/x509/pkix"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/pkg/errors"
)

// GrantOption
type GrantOption func(*Grant)

func GrantOptionGenerateFingerprint(data []byte) GrantOption {
	return func(g *Grant) {
		g.Fingerprint = sshx.FingerprintSHA256(data)
	}
}

func AgentGrant(pub []byte, options ...GrantOption) *Grant {
	g := Grant{
		Permission:    agent(),
		Authorization: pub,
	}

	for _, opt := range options {
		opt(&g)
	}

	return g.EnsureDefaults()
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
	Sync(ctx context.Context, b Bloomy, c chan *Grant) (err error)
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
	}.merge(options...)
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

	if resp.Authority, resp.PrivateKey, resp.Certificate, err = t.authority.Create(20*time.Hour, rsax.AutoBits(), caoptions...); err != nil {
		log.Println("failed to generate credentials", err)
		return nil, status.Error(codes.Unavailable, "authority not available")
	}

	return &resp, nil
}

// Search the notary service for grants.
func (t Service) Search(req *SearchRequest, s Notary_SearchServer) (err error) {
	if p := t.auth.Authorize(s.Context()); !p.Search {
		return status.Error(codes.PermissionDenied, "invalid credentials")
	}
	grantevent := func(grants []*Grant) *SearchResponse {
		return &SearchResponse{
			Grants: grants,
		}
	}

	b := bloom.NewWithEstimates(1000, 0.0001)

	out := make(chan *Grant, 200)
	errc := make(chan error)
	go func() {
		errc <- t.storage.Sync(s.Context(), b, out)
		close(out)
	}()

	batch := make([]*Grant, 0, 100)

	for {
		select {
		case g, ok := <-out:
			if !ok {
				if err = s.Send(grantevent(batch)); err != nil {
					log.Println(errors.Wrap(err, "failed to send event"))
					return status.Error(codes.Internal, "failed to send event")
				}
				return nil
			}

			if batch = append(batch, g); len(batch) < cap(batch) {
				continue
			}

			if err = s.Send(grantevent(batch)); err != nil {
				log.Println(errors.Wrap(err, "failed to send event"))
				return status.Error(codes.Internal, "failed to send event")
			}

			batch = batch[:0]
		case err := <-errc:
			if err == nil {
				continue
			}

			if cause := s.Send(grantevent(batch)); cause != nil {
				log.Println(errors.Wrap(cause, "failed to send event"))
			}

			return status.Error(codes.Internal, "failed to send event")
		}
	}
}
