package agentutil

import (
	"context"
	"log"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/logx"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// EnsureLeader waits for a leader to be established.
func EnsureLeader(ctx context.Context, d dialer, proxy agent.Peer) (info agent.InfoResponse, err error) {
	var (
		qc agent.Client
	)

	rl := rate.NewLimiter(rate.Every(10*time.Second), 1)

	for {
		if qc != nil {
			logx.MaybeLog(errors.Wrap(qc.Close(), "failed to close previous client"))
		}

		if err = rl.Wait(ctx); err != nil {
			log.Println(errors.Wrap(err, "failed to wait during rate limiting"))
			continue
		}

		if qc, err = agent.NewProxyDialer(d).Dial(proxy); err != nil {
			log.Println(errors.Wrap(err, "failed to dial quorum peer"))
			continue
		}

		if info, err = qc.QuorumInfo(); err != nil {
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
