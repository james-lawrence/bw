package agent

import (
	"context"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type auth interface {
	Deploy(ctx context.Context) error
}

type noauth struct{}

func (t noauth) Deploy(context.Context) error {
	return status.Error(codes.PermissionDenied, "invalid credentials")
}
