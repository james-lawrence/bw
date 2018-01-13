package agent

import "context"

type quorum interface {
	Deploy(concurrency int64, archive Archive, peers ...Peer) error
	Upload(stream Quorum_UploadServer) error
	Watch(stream Quorum_WatchServer) error
	Dispatch(in Quorum_DispatchServer) error
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
	return &_zero, t.q.Deploy(req.Concurrency, *req.Archive)
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
func (t Quorum) Dispatch(in Quorum_DispatchServer) error {
	return t.q.Dispatch(in)
}
