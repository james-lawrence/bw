package agentutil

import (
	"context"
	"log"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/errorsx"
	"google.golang.org/grpc"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// EnsureLeader waits for a leader to be established.
func EnsureLeader(ctx context.Context, d dialers.DefaultsDialer, proxy *agent.Peer) (info *agent.InfoResponse, err error) {
	var (
		conn *grpc.ClientConn
	)

	rl := rate.NewLimiter(rate.Every(10*time.Second), 1)
	pd := dialers.NewProxy(d)

	for {
		if conn != nil {
			errorsx.MaybeLog(errors.Wrap(conn.Close(), "failed to close previous client"))
		}

		if err = rl.Wait(ctx); err != nil {
			log.Println(errors.Wrap(err, "failed to wait during rate limiting"))
			continue
		}

		if conn, err = pd.DialContext(ctx); err != nil {
			log.Println(errors.Wrap(err, "failed to dial quorum peer"))
			continue
		}

		if info, err = agent.NewConn(conn).QuorumInfo(ctx); err != nil {
			log.Println(errors.Wrap(err, "failed to retrieve quorum information"))
			continue
		}

		if info.Leader == nil {
			log.Println("no leader has been elected")
			continue
		}

		return info, err
	}
}
