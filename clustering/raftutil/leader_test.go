package raftutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

func singleNodeLeader(t *testing.T) *raft.Raft {
	conf := raft.DefaultConfig()
	conf.LocalID = "local"
	conf.LogOutput = nil
	conf.HeartbeatTimeout = 50 * time.Millisecond
	conf.ElectionTimeout = 50 * time.Millisecond
	conf.LeaderLeaseTimeout = 50 * time.Millisecond
	conf.CommitTimeout = 5 * time.Millisecond

	store := raft.NewInmemStore()
	_, transport := raft.NewInmemTransport("local")
	r, err := raft.NewRaft(conf, &noopFSM{}, store, store, raft.NewInmemSnapshotStore(), transport)
	require.NoError(t, err)
	t.Cleanup(func() { r.Shutdown().Error() })

	require.NoError(t, r.BootstrapCluster(raft.Configuration{Servers: []raft.Server{{ID: "local", Address: transport.LocalAddr()}}}).Error())
	require.Eventually(t, func() bool { return r.State() == raft.Leader }, 5*time.Second, 10*time.Millisecond)
	return r
}

// a leader that can't transfer leadership (no other peers) must leave the protocol instead of
// remaining a leader that can't commit.
func TestLeaderStepdownShouldLeaveWhenTransferFails(t *testing.T) {
	var left sync.WaitGroup
	left.Add(1)

	meta := stateMeta{
		r:        singleNodeLeader(t),
		protocol: &Protocol{},
		done:     func() { left.Done() },
		ctx:      context.Background(),

		barrierTimeout:     defaultBarrierTimeout,
		maxBarrierFailures: defaultMaxBarrierFailures,
		stepdownTimeout:    defaultStepdownTimeout,
	}

	next := leader{stateMeta: meta, barrierFailures: meta.maxBarrierFailures}.stepdown(nil, context.DeadlineExceeded)
	require.IsType(t, passive{}, next)
	left.Wait() // done was invoked.
}
