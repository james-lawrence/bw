package grpcx

import (
	"log"
	"sync"

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

// IsUnimplemented checks if the endpoint is not implemented.
func IsUnimplemented(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.Unimplemented
	}

	return false
}

// IsUnauthorized checks if its a grpc unauthorized error..
func IsUnauthorized(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.PermissionDenied
	}

	return false
}

// NewCachedClient ...
func NewCachedClient() *CachedClient {
	return &CachedClient{m: &sync.RWMutex{}}
}

// CachedClient caches a grpc client for use, when no connection is cached
// it'll use the provided address and dial options to establish a connection.
type CachedClient struct {
	conn *grpc.ClientConn
	m    *sync.RWMutex
}

// Dial - returns the cached connection if any, otherwise it'll use the provided
// options to establish a connection.
func (t *CachedClient) Dial(addr string, options ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	t.m.RLock()
	c := t.conn
	t.m.RUnlock()

	if c != nil {
		return c, nil
	}

	t.m.Lock()
	defer t.m.Unlock()

	t.conn, err = grpc.Dial(addr, options...)
	return t.conn, err
}
