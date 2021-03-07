package daemons

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	_cluster "github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering/raftutil"
)

type rafter interface {
	Raft(ctx context.Context, conf agent.Config, sq raftutil.BacklogQueueWorker, options ...raftutil.ProtocolOption) (raftutil.Protocol, error)
}

// Quorum initialize the quorum daemon service.
func Quorum(dctx Context, cc rafter) (_ Context, err error) {
	transport := raftutil.ProtocolOptionMuxerTransport(dctx.Config.P2PBind, dctx.Muxer, dctx.RPCCredentials)
	sq := raftutil.BacklogQueueWorker{
		Queue: make(chan *agent.ClusterWatchEvents, 100),
	}

	go _cluster.NewEventsSubscription(dctx.Inmem, sq.Enqueue)

	if dctx.Raft, err = cc.Raft(dctx.Context, dctx.Config, sq, transport); err != nil {
		return dctx, err
	}

	return dctx, err
}
