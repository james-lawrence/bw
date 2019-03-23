package grpcx

import (
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IgnoreShutdownErrors ignores common (safe) shutdown errors.
func IgnoreShutdownErrors(err error) error {
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.Canceled, codes.Unavailable:
			return nil
		}

		log.Println("status error", s.Code(), s.Message())
	}

	if err == grpc.ErrClientConnClosing {
		return nil
	}

	return err
}
