package notary

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/james-lawrence/bw/internal/x/errorsx"
)

// PermAll all the permissions.
func PermAll() *Permission {
	return ptr(all())
}

// grant all permissions
func all() Permission {
	return Permission{
		Grant:   true,
		Revoke:  true,
		Search:  true,
		Refresh: true,
	}
}

// grant no permissions
func none() Permission {
	return Permission{}
}

func ptr(p Permission) *Permission {
	return &p
}

func unwrap(p *Permission) Permission {
	if p == nil {
		return none()
	}

	return *p
}

type storage interface {
	Lookup(fingerprint string) (g Grant, err error)
	Insert(Grant) (Grant, error)
	Delete(Grant) (Grant, error)
}

type option func(*Service)

// New notary service.
func New(s storage, options ...option) Service {
	return Service{
		storage: s,
		auth:    newAuth(s),
	}
}

// Service of a notary service
type Service struct {
	storage storage
	auth    auth
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
		g    Grant
		resp GrantResponse
	)

	if p := t.auth.Authorize(ctx); !p.Grant {
		return &resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if req.Grant == nil {
		return &resp, errorsx.String("a grant must be provided")
	}

	if g, err = t.storage.Insert(*req.Grant); err != nil {
		return &resp, err
	}

	return &GrantResponse{
		Grant: &g,
	}, nil
}

// Revoke a grant from the notary service.
func (t Service) Revoke(ctx context.Context, req *RevokeRequest) (_ *RevokeResponse, err error) {
	var (
		g    Grant
		resp RevokeResponse
	)

	if p := t.auth.Authorize(ctx); !p.Revoke {
		return &resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if g, err = t.storage.Delete(Grant{Fingerprint: req.Fingerprint}); err != nil {
		return &resp, err
	}

	return &RevokeResponse{
		Grant: &g,
	}, nil
}

// Refresh generate new TLS credentials
func (t Service) Refresh(ctx context.Context, req *RefreshRequest) (_ *RefreshResponse, err error) {
	var (
		resp RefreshResponse
	)

	log.Println("Notary.Refresh invoked")
	if p := t.auth.Authorize(ctx); !p.Refresh {
		return &resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	return &resp, status.Error(codes.Unimplemented, "")
}

// Search the notary service for grants.
func (t Service) Search(req *SearchRequest, dst Notary_SearchServer) (err error) {
	if p := t.auth.Authorize(dst.Context()); !p.Search {
		return status.Error(codes.PermissionDenied, "invalid credentials")
	}

	return errorsx.String("not implemented")
}
