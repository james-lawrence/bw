package grpcx

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"sync"
	"time"

	"github.com/akutz/memconn"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// DebugIntercepter prints when each request is initiated and completed
func DebugIntercepter(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	var (
		addr net.Addr
	)

	if x, ok := peer.FromContext(ctx); ok {
		addr = x.Addr
	}

	log.Printf("%T %s client(%s) initiated\n", info.Server, info.FullMethod, addr.String())
	defer log.Printf("%T %s client(%s) completed\n", info.Server, info.FullMethod, addr.String())

	return handler(ctx, req)
}

// DebugStreamIntercepter prints when each stream is initiated and completed
func DebugStreamIntercepter(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	var (
		addr net.Addr
	)

	if x, ok := peer.FromContext(ss.Context()); ok {
		addr = x.Addr
	}

	log.Printf("%T %s client(%s) initiated\n", srv, info.FullMethod, addr.String())
	defer log.Printf("%T %s client(%s) completed\n", srv, info.FullMethod, addr.String())
	return handler(srv, ss)
}

// DebugClientIntercepter prints each rpc invoked.
func DebugClientIntercepter(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	log.Printf("%s%s initiated\n", cc.Target(), method)
	defer log.Printf("%s%s completed\n", cc.Target(), method)
	return invoker(ctx, method, req, reply, cc, opts...)
}

// DebugClientStreamIntercepter prints each stream invocation.
func DebugClientStreamIntercepter(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	log.Printf("%s%s stream initiated\n", cc.Target(), method)
	defer log.Printf("%s%s stream completed\n", cc.Target(), method)
	return streamer(ctx, desc, cc, method, opts...)
}

// InsecureTLS generate insecure transport credentials.
func InsecureTLS() credentials.TransportCredentials {
	return credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
}

// IgnoreShutdownErrors ignores common (safe) shutdown errors.
func IgnoreShutdownErrors(err error) error {
	if s, ok := status.FromError(errors.Unwrap(err)); ok {
		switch s.Code() {
		case codes.Canceled, codes.Unavailable:
			return nil
		}
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

// IsUnauthorized checks if its a grpc unauthorized error.
func IsUnauthorized(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.PermissionDenied
	}

	return false
}

// IsUnavailable ...
func IsUnavailable(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.Unavailable
	}

	return false
}

// Retry the given function if error matches a grpc code.
func Retry(op func() error, retry ...codes.Code) (err error) {
	l := rate.NewLimiter(rate.Every(time.Second), 1)
	for {
		if err = op(); !IsCode(err, retry...) {
			return err
		}

		l.Wait(context.Background())
	}
}

func IsCode(err error, check ...codes.Code) bool {
	if s, ok := status.FromError(errors.Cause(err)); ok {
		for _, c := range check {
			if s.Code() == c {
				return true
			}
		}
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

func DialInmem() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, address string) (conn net.Conn, err error) {
		return memconn.DialContext(ctx, "memu", address)
	})
}
