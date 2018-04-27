package agent

import (
	"context"
)

type quorum interface {
	Deploy(opts DeployOptions, a Archive, peers ...Peer) error
	Upload(stream Quorum_UploadServer) error
	Watch(stream Quorum_WatchServer) error
	Dispatch(context.Context, ...Message) error
}

// NewQuorum ...
func NewQuorum(q quorum) Quorum {
	return Quorum{
		q: q,
	}
}

// Quorum implements quorum functionality.
type Quorum struct {
	q quorum
}

// Deploy ...
func (t Quorum) Deploy(ctx context.Context, req *DeployCommandRequest) (_ *DeployCommandResult, err error) {
	var (
		_zero DeployCommandResult
	)
	return &_zero, t.q.Deploy(*req.Options, *req.Archive, PtrToPeers(req.Peers...)...)
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
	return &DispatchResponse{}, t.q.Dispatch(ctx, MessagesFromPtr(req.Messages...)...)
}
