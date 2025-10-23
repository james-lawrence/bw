package proxy

import (
	"context"
	"io"
	"log"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/notary"
)

type auth interface {
	Authorize(ctx context.Context) *notary.Permission
}

// NewDeployment proxy for deploys, dialer must be a quorum dialer.
func NewDeployment(a auth, d dialers.Quorum) Deployment {
	return Deployment{Auth: a, Dialer: d}
}

// Deployment - proxy deployment commands from any agent to quorum.
type Deployment struct {
	agent.UnimplementedDeploymentsServer
	Auth   auth
	Dialer dialers.Quorum
}

// Bind to the given grpc server.
func (t Deployment) Bind(s *grpc.Server) {
	agent.RegisterDeploymentsServer(s, t)
}

func (t Deployment) conn(ctx context.Context) (conn *grpc.ClientConn, err error) {
	return t.Dialer.DialContext(ctx)
}

func (t Deployment) direct(ctx context.Context, p *agent.Peer) (conn *grpc.ClientConn, err error) {
	return dialers.NewDirect(agent.RPCAddress(p), t.Dialer.Defaults()...).DialContext(ctx)
}

// Upload a deployment archive into the cluster
func (t Deployment) Upload(stream agent.Deployments_UploadServer) (err error) {
	var (
		cc     *grpc.ClientConn
		upload agent.Quorum_UploadClient
		chunk  *agent.UploadChunk
		resp   *agent.UploadResponse
	)

	if !t.Auth.Authorize(stream.Context()).Deploy {
		return status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if cc, err = t.conn(stream.Context()); err != nil {
		log.Println("proxy connection failed", err)
		return status.Error(codes.Unavailable, "proxy connection error")
	}
	defer cc.Close()

	if upload, err = agent.NewQuorumClient(cc).Upload(stream.Context()); err != nil {
		return err
	}

	for err == nil {
		if chunk, err = stream.Recv(); err != nil {
			continue
		}

		if err = upload.Send(chunk); err != nil {
			continue
		}
	}

	if err != io.EOF {
		return err
	}

	if resp, err = upload.CloseAndRecv(); err != nil {
		return err
	}

	return stream.SendAndClose(resp)
}

// Deploy execute an actual deployment archive.
func (t Deployment) Deploy(ctx context.Context, req *agent.DeployCommandRequest) (resp *agent.DeployCommandResult, err error) {
	var (
		cc *grpc.ClientConn
	)

	if !t.Auth.Authorize(ctx).Deploy {
		return resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if cc, err = t.conn(ctx); err != nil {
		cause := status.Error(codes.Unavailable, "proxy connection error")
		errorsx.Log(cause)
		return resp, cause
	}
	defer cc.Close()

	return agent.NewQuorumClient(cc).Deploy(ctx, req)
}

// Cancel an active deploy.
func (t Deployment) Cancel(ctx context.Context, req *agent.CancelRequest) (resp *agent.CancelResponse, err error) {
	var (
		cc *grpc.ClientConn
	)

	if !t.Auth.Authorize(ctx).Deploy {
		return resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if cc, err = t.conn(ctx); err != nil {
		cause := status.Error(codes.Unavailable, "proxy connection error")
		errorsx.Log(cause)
		return resp, cause
	}
	defer cc.Close()

	return agent.NewQuorumClient(cc).Cancel(ctx, req)
}

// Watch watch for events.
func (t Deployment) Watch(req *agent.WatchRequest, out agent.Deployments_WatchServer) (err error) {
	var (
		cc *grpc.ClientConn
		w  agent.Quorum_WatchClient
	)

	if !t.Auth.Authorize(out.Context()).Deploy {
		return status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if cc, err = t.conn(out.Context()); err != nil {
		cause := status.Error(codes.Unavailable, "proxy connection error")
		errorsx.Log(cause)
		return cause
	}
	defer cc.Close()

	if w, err = agent.NewQuorumClient(cc).Watch(out.Context(), req); err != nil {
		log.Println("watch quorum client failed", err)
		return err
	}

	for msg, err := w.Recv(); err == nil; msg, err = w.Recv() {
		if err = out.Send(msg); err != nil {
			return err
		}
	}

	return errorsx.Compact(errors.WithStack(err), w.CloseSend())
}

// Dispatch messages to the state machine.
func (t Deployment) Dispatch(ctx context.Context, req *agent.DispatchRequest) (resp *agent.DispatchResponse, err error) {
	var (
		cc *grpc.ClientConn
	)

	if !t.Auth.Authorize(ctx).Deploy {
		return resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if cc, err = t.conn(ctx); err != nil {
		cause := status.Error(codes.Unavailable, "proxy connection error")
		errorsx.Log(cause)
		return resp, cause
	}
	defer cc.Close()

	return agent.NewQuorumClient(cc).Dispatch(ctx, req)
}

// Logs retrieve logs for the given deploy.
func (t Deployment) Logs(req *agent.LogRequest, out agent.Deployments_LogsServer) (err error) {
	var (
		cc *grpc.ClientConn
		w  agent.Deployments_LogsClient
	)

	if !t.Auth.Authorize(out.Context()).Deploy {
		return status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if req.Peer == nil {
		return status.Error(
			codes.FailedPrecondition,
			"peer to retrieve logs from required",
		)
	}

	if cc, err = t.direct(out.Context(), req.Peer); err != nil {
		cause := status.Error(codes.Unavailable, "proxy connection error")
		errorsx.Log(cause)
		return cause
	}
	defer cc.Close()

	if w, err = agent.NewAgentClient(cc).Logs(out.Context(), req); err != nil {
		return err
	}

	for msg, err := w.Recv(); err == nil; msg, err = w.Recv() {
		if err = out.Send(msg); err != nil {
			return err
		}
	}

	return errorsx.Compact(errors.WithStack(err), w.CloseSend())
}
