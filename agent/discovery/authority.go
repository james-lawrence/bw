package discovery

import (
	"context"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/james-lawrence/bw/agent"
)

// NewAuthority service.
func NewAuthority(protoPath string) Authority {
	return Authority{protoPath: protoPath}
}

// Authority provides methods around the TLS authority.
type Authority struct {
	UnimplementedAuthorityServer
	protoPath string
}

// Bind to the grpc server.
func (t Authority) Bind(s *grpc.Server) {
	RegisterAuthorityServer(s, t)
}

// Check the fingerprint against the authority.
func (t Authority) Check(ctx context.Context, req *CheckRequest) (resp *CheckResponse, err error) {
	var (
		m1      agent.TLSCertificates
		encoded []byte
	)

	if encoded, err = os.ReadFile(t.protoPath); err != nil {
		return &CheckResponse{}, status.Error(codes.Unavailable, "missing info")
	}

	if err = proto.Unmarshal(encoded, &m1); err != nil {
		return &CheckResponse{}, status.Error(codes.Unavailable, "invalid authority")
	}

	if m1.Fingerprint != req.Fingerprint {
		return &CheckResponse{}, status.Error(codes.NotFound, "fingerprint mismatch")
	}

	return &CheckResponse{}, nil
}
