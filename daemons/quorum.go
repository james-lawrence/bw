package daemons

import (
	"context"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"google.golang.org/grpc"
)

type rafter interface {
	Raft(ctx context.Context, conf agent.Config, node *memberlist.Node, eq *grpc.ClientConn, options ...raftutil.ProtocolOption) (raftutil.Protocol, error)
}

// Quorum initialize the quorum daemon service.
func Quorum(dctx Context, cc rafter) (_ Context, err error) {
	transport := raftutil.ProtocolOptionMuxerTransport(dctx.Config.P2PBind, dctx.Config.P2PAdvertised, dctx.Muxer, raftutil.NewTLSStreamDialer(dctx.RPCCredentials))

	if dctx.Raft, err = cc.Raft(dctx.Context, dctx.Config, dctx.Cluster.LocalNode(), dctx.Inmem, transport); err != nil {
		return dctx, err
	}

	return dctx, err
}
