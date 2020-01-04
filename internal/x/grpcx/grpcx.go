package grpcx

import (
	"context"
	"crypto/tls"
	"log"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// DebugIntercepter prints when each request is initiated and completed
func DebugIntercepter(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	log.Printf("%T %s initiated", info.Server, info.FullMethod)
	defer log.Printf("%T %s completed", info.Server, info.FullMethod)
	return handler(ctx, req)
}

// DebugStreamIntercepter prints when each stream is initiated and completed
func DebugStreamIntercepter(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Printf("%T %s initiated", srv, info.FullMethod)
	defer log.Printf("%T %s completed", srv, info.FullMethod)
	return handler(srv, ss)
}

// InsecureTLS generate insecure transport credentials.
func InsecureTLS() credentials.TransportCredentials {
	return credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
}

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

// IsNotFound check if the error is a grpc not found status error.
func IsNotFound(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.NotFound
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

// Close close the cached connection.
func (t *CachedClient) Close() error {
	t.m.RLock()
	c := t.conn
	t.m.RUnlock()

	if c == nil {
		return nil
	}

	return c.Close()
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
