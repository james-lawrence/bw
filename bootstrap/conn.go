package bootstrap

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Latest get the latest archive from the bootstrap socket.
func Latest(ctx context.Context, uds string, options ...grpc.DialOption) (latest *agent.Deploy, err error) {
	var (
		c Conn
	)

	if c, err = dial(ctx, uds, options...); err != nil {
		return latest, err
	}

	defer c.conn.Close()

	return c.Archive(ctx)
}

func getfallback(c agent.Config, options ...grpc.DialOption) (latest *agent.Deploy, err error) {
	const done = errorsx.String("done")
	var (
		compacted error = agentutil.ErrNoDeployments
	)

	qs := SocketQuorum(c)
	ls := SocketLocal(c)

	err = filepath.Walk(root(c), func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		switch path {
		case qs:
			return nil
		case ls:
			return nil
		default:

			if latest, err = Latest(context.Background(), path, options...); err == nil {
				return done
			}

			if !agentutil.IsNoDeployments(err) {
				compacted = err
			}

			log.Println("bootstrap failed", err)
			return nil
		}
	})

	if err != done {
		return latest, compacted
	}

	return latest, nil
}

// dial connect to a unix domain socket.
func dial(ctx context.Context, uds string, options ...grpc.DialOption) (c Conn, err error) {
	var (
		conn *grpc.ClientConn
	)

	options = append(options,
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", addr)
		}),
		// grpc.WithUnaryInterceptor(grpcx.DebugClientIntercepter),
		// grpc.WithStreamInterceptor(grpcx.DebugClientStreamIntercepter),
	)

	if conn, err = grpc.DialContext(ctx, uds, options...); err != nil {
		return c, err
	}

	return Conn{conn: conn}, nil
}

// Conn connection to the bootstrap service
type Conn struct {
	conn *grpc.ClientConn
}

// Archive request the latest archive from the service.
func (t Conn) Archive(ctx context.Context) (latest *agent.Deploy, err error) {
	var (
		req  agent.ArchiveRequest
		resp *agent.ArchiveResponse
	)

	rpc := agent.NewBootstrapClient(t.conn)

	if resp, err = rpc.Archive(ctx, &req); err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return latest, agentutil.ErrNoDeployments
		default:
			return latest, errors.Wrapf(err, "bootstrap service: %s", t.conn.Target())
		}
	}

	if resp == nil {
		return latest, errors.New("invalid response from bootstrap service")
	}

	switch resp.Info {
	case agent.ArchiveResponse_ActiveDeploy:
		return resp.Deploy, agentutil.ErrActiveDeployment
	default:
		return resp.Deploy, nil
	}
}
