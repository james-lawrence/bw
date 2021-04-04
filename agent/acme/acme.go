// Package acme implements the acme protocol. specifically for the alpn for the cluster.
// this forces a couple requirements, the discovery service must be exposed on port 443.
// another reference implementation can be seen at:
// https://github.com/caddyserver/caddy/pull/2201/files
package acme

import (
	context "context"
	"log"

	"github.com/hashicorp/memberlist"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/notary"
)

type rendezvous interface {
	Get([]byte) *memberlist.Node
}

// ReadConfig ...
func ReadConfig(c agent.Config, path string) (svc DiskCache, err error) {
	type config struct {
		ACME certificatecache.ACMEConfig `yaml:"acme"`
	}

	var (
		cc = &config{
			ACME: certificatecache.DefaultACMEConfig(),
		}
	)

	if err = bw.ExpandAndDecodeFile(path, cc); err != nil {
		return svc, err
	}

	a := account{ACMEConfig: cc.ACME, Config: c}

	return newDiskcache(c, cc.ACME, a), nil
}

type auth interface {
	Authorize(ctx context.Context) *notary.Permission
}

// NewService new acme service from an agent.Configuration.
func NewService(cache DiskCache, a auth) Server {
	return Server{
		cache: cache,
		auth:  a,
	}
}

// Server is responsible for generating and resolving ACME protocol certificates.
type Server struct {
	UnimplementedACMEServer
	cache DiskCache
	auth  auth
}

// Challenge solve the challenge.
func (t Server) Challenge(ctx context.Context, req *ChallengeRequest) (resp *ChallengeResponse, err error) {
	if !t.auth.Authorize(ctx).Autocert {
		return resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	return t.cache.Challenge(ctx, req)
}

// Resolution return the resolution
func (t Server) Resolution(ctx context.Context, req *ResolutionRequest) (resp *ResolutionResponse, err error) {
	log.Println("retrieving response to challenge")
	if !t.auth.Authorize(ctx).Autocert {
		return resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	return t.cache.Resolution(ctx, req)
}
