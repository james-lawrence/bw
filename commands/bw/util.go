package main

import (
	"context"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// WatchEvents pushes events into the provided channel for the given cluster.
func WatchEvents(d agent.Dialer, c cluster.Cluster, events chan agent.Message) {
	rl := rate.NewLimiter(rate.Every(time.Second), 1)
	for {
		var (
			err   error
			qc    agent.Client
			local = c.Local()
		)

		if err = rl.Wait(context.Background()); err != nil {
			events <- agentutil.LogError(local, errors.Wrap(err, "failed to wait during rate limiting"))
			continue
		}

		if qc, err = agent.NewQuorumDialer(d).Dial(c); err != nil {
			events <- agentutil.LogError(local, errors.Wrap(err, "events dialer failed to connect"))
			continue
		}

		if err = qc.Watch(events); err != nil {
			events <- agentutil.LogError(local, errors.Wrap(err, "connection lost, reconnecting"))
			continue
		}
	}
}
