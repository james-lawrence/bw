package bootstrap_test

import (
	"context"
	"io"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/storage"
	"github.com/pkg/errors"
)

type fakeClient struct {
	errResult error
	archive   agent.Archive
	deploy    agent.Deploy
	connect   agent.ConnectResponse
	status    agent.StatusResponse
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

func (t fakeClient) Upload(initiator string, srcbytes uint64, src io.Reader) (agent.Archive, error) {
	return t.archive, t.errResult
}

func (t fakeClient) RemoteDeploy(dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) error {
	return t.errResult
}

func (t fakeClient) Deploy(agent.DeployOptions, agent.Archive) (agent.Deploy, error) {
	return t.deploy, t.errResult
}

func (t fakeClient) Connect() (agent.ConnectResponse, error) {
	return t.connect, t.errResult
}

func (t fakeClient) Info() (agent.StatusResponse, error) {
	return t.status, t.errResult
}

func (t fakeClient) Watch(out chan<- agent.Message) error {
	return t.errResult
}

func (t fakeClient) Dispatch(messages ...agent.Message) error {
	return t.errResult
}

type fakeDialer struct {
	c fakeClient
}

func (t fakeDialer) Dial(p agent.Peer) (agent.Client, error) {
	return t.c, nil
}

type noopDeployer struct {
	err error
}

func (t noopDeployer) Deploy(dctx deployment.DeployContext) {
	dctx.Done(t.err)
}

var _ = Describe("Bootstrap", func() {
	It("should succeed when no errors occur", func() {
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{
					{
						Archive: &agent.Archive{
							Peer:         &p,
							Ts:           time.Now().Unix(),
							DeploymentID: bw.MustGenerateID(),
						},
					},
				},
			},
		}

		fd := fakeDialer{c: fc}
		mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionQuiet(),
		)
		Expect(Bootstrap(context.Background(), p, mc, fd, dc)).ToNot(HaveOccurred())
	})

	It("should fail when it fails to download the archive", func() {
		reg := storage.NoopRegistry{Err: errors.New("download failed")}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{
					{
						Archive: &agent.Archive{
							Peer:         &p,
							Ts:           time.Now().Unix(),
							DeploymentID: bw.MustGenerateID(),
						},
					},
				},
			},
		}

		fd := fakeDialer{c: fc}
		mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionQuiet(),
		)
		Expect(errors.Cause(Bootstrap(context.Background(), p, mc, fd, dc))).To(MatchError("download failed"))
	})

	It("should fail when the deployment fails", func() {
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{
					{
						Archive: &agent.Archive{
							Peer:         &p,
							Ts:           time.Now().Unix(),
							DeploymentID: bw.MustGenerateID(),
						},
					},
				},
			},
		}

		fd := fakeDialer{c: fc}
		mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
		dc := deployment.New(
			p,
			noopDeployer{err: errors.New("deployment failed")},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionQuiet(),
		)
		Expect(errors.Cause(Bootstrap(context.Background(), p, mc, fd, dc))).To(MatchError("deployment failed"))
	})
})
