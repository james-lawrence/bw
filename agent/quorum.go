package agent

import (
	"context"

	"github.com/james-lawrence/bw/internal/grpcx"
	"google.golang.org/grpc"
)

type quorum interface {
	Deploy(ctx context.Context, by string, opts *DeployOptions, a *Archive, peers ...*Peer) error
	Upload(stream Quorum_UploadServer) error
	Watch(stream Quorum_WatchServer) error
	History(context.Context) ([]*Message, error)
	Dispatch(context.Context, ...*Message) error
	Info(context.Context) (InfoResponse, error)
	Cancel(context.Context, *CancelRequest) error
}

// NewQuorum ...
func NewQuorum(q quorum, a auth) Quorum {
	return Quorum{
		q:    q,
		auth: a,
	}
}

// Quorum implements quorum functionality.
type Quorum struct {
	UnimplementedQuorumServer
	auth
	q quorum
}

// Bind to a grpc server.
func (t Quorum) Bind(srv *grpc.Server) Quorum {
	RegisterQuorumServer(srv, t)
	return t
}

// Info ...
func (t Quorum) Info(ctx context.Context, _ *InfoRequest) (_ *InfoResponse, err error) {
	var (
		resp InfoResponse
	)

	if err = t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	resp, err = t.q.Info(ctx)

	return &resp, err
}

// Deploy ...
func (t Quorum) Deploy(ctx context.Context, req *DeployCommandRequest) (_ *DeployCommandResult, err error) {
	if err := t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	if err = t.q.Deploy(ctx, req.Initiator, req.Options, req.Archive, req.Peers...); grpcx.IsUnavailable(err) {
		return nil, err
	}

	return &DeployCommandResult{}, err
}

// History
func (t Quorum) History(ctx context.Context, req *HistoryRequest) (resp *HistoryResponse, err error) {
	var (
		history []*Message
	)
	if err := t.auth.Deploy(ctx); err != nil {
		return resp, err
	}

	if history, err = t.q.History(ctx); err != nil {
		return resp, err
	}

	return &HistoryResponse{
		Messages: history,
	}, nil
}

// Upload ...
func (t Quorum) Upload(stream Quorum_UploadServer) (err error) {
	if err := t.auth.Deploy(stream.Context()); err != nil {
		return err
	}

	return t.q.Upload(stream)
}

// Watch watch for events.
func (t Quorum) Watch(_ *WatchRequest, out Quorum_WatchServer) (err error) {
	if err := t.auth.Deploy(out.Context()); err != nil {
		return err
	}

	return t.q.Watch(out)
}

// Dispatch record deployment events.
func (t Quorum) Dispatch(ctx context.Context, req *DispatchRequest) (*DispatchResponse, error) {
	if err := t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	return &DispatchResponse{}, t.q.Dispatch(ctx, req.Messages...)
}

// Cancel the active deploy.
func (t Quorum) Cancel(ctx context.Context, req *CancelRequest) (*CancelResponse, error) {
	if err := t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	return &CancelResponse{}, t.q.Cancel(ctx, req)
}
