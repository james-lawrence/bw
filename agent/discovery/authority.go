package discovery

import (
	"context"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
		ok      bool
		m1      agent.Message
		m2      *agent.Message_Authority
		evt     agent.TLSEvent
		encoded []byte
	)

	if encoded, err = ioutil.ReadFile(t.protoPath); err != nil {
		return &CheckResponse{}, status.Error(codes.Unavailable, "missing info")
	}

	if err = proto.Unmarshal(encoded, &m1); err != nil {
		return &CheckResponse{}, status.Error(codes.Unavailable, "invalid authority")
	}

	if m2, ok = m1.GetEvent().(*agent.Message_Authority); !ok {
		return &CheckResponse{}, status.Error(codes.Unavailable, "invalid authority")
	}

	if m2 == nil {
		return &CheckResponse{}, status.Error(codes.Unavailable, "invalid authority")
	}

	evt = *m2.Authority

	if evt.Fingerprint != req.Fingerprint {
		return &CheckResponse{}, status.Error(codes.NotFound, "fingerprint mismatch")
	}

	return &CheckResponse{}, nil
}
