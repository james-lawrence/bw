package bootstrap_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"

	"google.golang.org/grpc"
)

// Mock bootstrap service
type Mock struct {
	agent.UnimplementedBootstrapServer
	Fail    error
	Current agent.Deploy
	Info    agent.ArchiveResponse_Info
}

// Archive - implements the bootstrap service.
func (t Mock) Archive(ctx context.Context, req *agent.ArchiveRequest) (resp *agent.ArchiveResponse, err error) {
	return &agent.ArchiveResponse{
		Info:   t.Info,
		Deploy: &t.Current,
	}, t.Fail
}

type fakeClient struct {
	errResult error
	archive   agent.Archive
	deploy    agent.Deploy
	connect   agent.ConnectResponse
	status    agent.StatusResponse
	qinfo     agent.InfoResponse
}

func (t fakeClient) Conn() *grpc.ClientConn {
	return nil
}

func (t fakeClient) Shutdown() error {
	return t.errResult
}

func (t fakeClient) Close() error {
	return t.errResult
}

func (t fakeClient) Cancel() error {
	return t.errResult
}

func (t fakeClient) NodeCancel() error {
	return t.errResult
}

func (t fakeClient) Logs(ctx context.Context, p *agent.Peer, did []byte) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(fmt.Sprintf("INFO: %s", string(did))))
}

func (t fakeClient) Upload(initiator string, srcbytes uint64, src io.Reader) (agent.Archive, error) {
	return t.archive, t.errResult
}

func (t fakeClient) RemoteDeploy(dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) error {
	return t.errResult
}

func (t fakeClient) Deploy(*agent.DeployOptions, *agent.Archive) (*agent.Deploy, error) {
	return &t.deploy, t.errResult
}

func (t fakeClient) Connect() (agent.ConnectResponse, error) {
	return t.connect, t.errResult
}

func (t fakeClient) Info() (agent.StatusResponse, error) {
	return t.status, t.errResult
}

func (t fakeClient) QuorumInfo() (agent.InfoResponse, error) {
	return t.qinfo, t.errResult
}

func (t fakeClient) Watch(_ context.Context, out chan<- *agent.Message) error {
	return t.errResult
}

func (t fakeClient) Dispatch(_ context.Context, messages ...*agent.Message) error {
	return t.errResult
}

type fakeDialer struct {
	c     fakeClient
	local fakeClient
}

func (t fakeDialer) Dial(...grpc.DialOption) (*grpc.ClientConnInterface, error) {
	if p.Name == "local" {
		return t.local, nil
	}
	return t.c, nil
}

func (t fakeDialer) Defaults(...grpc.DialOption) []grpc.DialOption {
	return []grpc.DialOption{}
}

type noopDeployer struct {
	err error
}

func (t noopDeployer) Deploy(dctx deployment.DeployContext) {
	dctx.Done(t.err)
}
