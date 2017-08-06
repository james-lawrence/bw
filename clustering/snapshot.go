package clustering

import (
	"context"
	"log"
	"time"
)

// SnapshotOption options for snapshotting.
type SnapshotOption func(*snapshot)

// SnapshotOptionContext specify the context to use for signalling shutdown.
func SnapshotOptionContext(ctx context.Context) SnapshotOption {
	return func(s *snapshot) {
		s.Context = ctx
	}
}

// SnapshotOptionFrequency specify how often a snapshot should be taken.
func SnapshotOptionFrequency(freq time.Duration) SnapshotOption {
	return func(s *snapshot) {
		s.Frequency = freq
	}
}

type snapshot struct {
	Context   context.Context
	Frequency time.Duration
}

func newSnapshot(options ...SnapshotOption) (snapper snapshot) {
	snapper = snapshot{
		Frequency: time.Hour,
		Context:   context.Background(),
	}
	for _, opt := range options {
		opt(&snapper)
	}
	return snapper
}

// Snapshot - performs a periodic snapshot of the cluster. blocking.
func Snapshot(c cluster, s snapshotter, options ...SnapshotOption) {
	var (
		err     error
		snapper = newSnapshot(options...)
	)
	take := func() {
		log.Println("taking snapshot of the cluster")
		if err = s.Snapshot(Peers(c)); err != nil {
			log.Println("failed to snapshot cluster", err)
		}
	}
	// take an initial snapshot immediately.
	take()
	// then take a snapshot every period.
	tick := time.NewTicker(snapper.Frequency)
	defer tick.Stop()
	for {
		select {
		case <-snapper.Context.Done():
			return
		case <-tick.C:
			take()
		}
	}
}
