package agent

import (
	"context"
)

type quorum interface {
	Deploy(opts DeployOptions, a Archive, peers ...*Peer) error
	Upload(stream Quorum_UploadServer) error
	Watch(stream Quorum_WatchServer) error
	Dispatch(context.Context, ...*Message) error
	Info(context.Context) (InfoResponse, error)
	Cancel(context.Context, *CancelRequest) error
}

// NewQuorum ...
func NewQuorum(q quorum) Quorum {
	return Quorum{
		q: q,
	}
}

// Quorum implements quorum functionality.
type Quorum struct {
	UnimplementedQuorumServer
	q quorum
}

// Info ...
func (t Quorum) Info(ctx context.Context, _ *InfoRequest) (_ *InfoResponse, err error) {
	var (
		resp InfoResponse
	)

	resp, err = t.q.Info(ctx)

	return &resp, err
}

// Deploy ...
func (t Quorum) Deploy(ctx context.Context, req *DeployCommandRequest) (_ *DeployCommandResult, err error) {
	var (
		_zero DeployCommandResult
	)

	return &_zero, t.q.Deploy(*req.Options, *req.Archive, req.Peers...)
}

// Upload ...
func (t Quorum) Upload(stream Quorum_UploadServer) (err error) {
	return t.q.Upload(stream)
}

// Watch watch for events.
func (t Quorum) Watch(_ *WatchRequest, out Quorum_WatchServer) (err error) {
	return t.q.Watch(out)
}

// Dispatch record deployment events.
func (t Quorum) Dispatch(ctx context.Context, req *DispatchRequest) (*DispatchResponse, error) {
	return &DispatchResponse{}, t.q.Dispatch(ctx, req.Messages...)
}

// Cancel the active deploy.
func (t Quorum) Cancel(ctx context.Context, req *CancelRequest) (*CancelResponse, error) {
	return &CancelResponse{}, t.q.Cancel(ctx, req)
}
