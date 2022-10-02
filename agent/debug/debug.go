package debug

import (
	"bytes"
	context "context"
	"log"

	"github.com/james-lawrence/bw/internal/bytesx"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/pkg/errors"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type auth interface {
	Deploy(ctx context.Context) error
}

func NewService(a auth) Service {
	return Service{
		auth: a,
	}
}

type Service struct {
	UnimplementedDebugServer
	auth auth
}

// Bind to a grpc server.
func (t Service) Bind(srv *grpc.Server) Service {
	RegisterDebugServer(srv, t)
	return t
}

func (t Service) Stacktrace(ctx context.Context, _ *StacktraceRequest) (_ *StacktraceResponse, err error) {
	if err = t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, 16*bytesx.KiB))

	if err = debugx.DumpRoutinesInto(iox.WriteNopCloser(buf)); err != nil {
		log.Println(errors.Wrap(err, "unable to generate stack trace"))
		return nil, status.Error(codes.Internal, "unable to generate stack trace")
	}

	return &StacktraceResponse{
		Trace: buf.Bytes(),
	}, nil
}
