package bootstrap_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/backoff"
	. "github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/storage"
	"github.com/james-lawrence/bw/x/testingx"
	"github.com/pkg/errors"
)

type fakeClient struct {
	errResult error
	archive   agent.Archive
	deploy    agent.Deploy
	connect   agent.ConnectResponse
	status    agent.StatusResponse
	qinfo     agent.InfoResponse
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

func (t fakeClient) Logs(did []byte) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(fmt.Sprintf("INFO: %s", string(did))))
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

func (t fakeClient) QuorumInfo() (agent.InfoResponse, error) {
	return t.qinfo, t.errResult
}

func (t fakeClient) Watch(_ context.Context, out chan<- agent.Message) error {
	return t.errResult
}

func (t fakeClient) Dispatch(_ context.Context, messages ...agent.Message) error {
	return t.errResult
}

type fakeDialer struct {
	c     fakeClient
	local fakeClient
}

func (t fakeDialer) Dial(p agent.Peer) (agent.Client, error) {
	if p.Name == "local" {
		return t.local, nil
	}
	return t.c, nil
}

type noopDeployer struct {
	err error
}

func (t noopDeployer) Deploy(dctx deployment.DeployContext) {
	dctx.Done(t.err)
}

var _ = Describe("Bootstrap", func() {
	var (
		peer1    = agent.NewPeer("node1")
		archive1 = agent.Archive{
			Peer:         &peer1,
			Ts:           time.Now().Unix(),
			DeploymentID: bw.MustGenerateID(),
		}
		archive2 = agent.Archive{
			Peer:         &peer1,
			Ts:           time.Now().Unix(),
			DeploymentID: bw.MustGenerateID(),
		}
		dopts1 = agent.DeployOptions{
			Timeout:           int64(time.Hour),
			SilenceDeployLogs: true,
		}
	)

	It("should succeed when no errors occur", func() {
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{
					{
						Archive: &archive1,
						Options: &dopts1,
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
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(Bootstrap(p, mc, fd, dc)).ToNot(HaveOccurred())
	})

	It("should fail when it fails to download the archive", func() {
		reg := storage.NoopRegistry{Err: errors.New("download failed")}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{
					// the latest deploy needs to be a failure to trigger the latest deploy check to fail.
					{
						Stage:   agent.Deploy_Failed,
						Archive: &archive1,
						Options: &dopts1,
					},
					{
						Stage:   agent.Deploy_Completed,
						Archive: &archive2,
						Options: &dopts1,
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
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(errors.Cause(Bootstrap(p, mc, fd, dc))).To(MatchError("download failed"))
	})

	It("should fail when the deployment fails", func() {
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{
					// the latest deploy needs to be a failure to trigger the latest deploy check to fail.
					{
						Stage:   agent.Deploy_Failed,
						Archive: &archive2,
						Options: &dopts1,
					},
					{
						Stage:   agent.Deploy_Completed,
						Archive: &archive1,
						Options: &dopts1,
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
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(errors.Cause(Bootstrap(p, mc, fd, dc))).To(MatchError("deployment failed"))
	})

	It("should succeed when it finishes bootstrapping from quorum", func() {
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")
		fc := fakeClient{
			qinfo: agent.InfoResponse{
				Mode: agent.InfoResponse_None,
				Deployed: &agent.DeployCommand{
					Command: agent.DeployCommand_Done,
					Archive: &archive1,
					Options: &dopts1,
				},
			},
		}

		fd := fakeDialer{c: fc}
		mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir()),
		)
		Expect(Bootstrap(p, mc, fd, dc)).ToNot(HaveOccurred())
	})

	Context("active deploy", func() {
		It("should deploy an active deploy from quorum", func() {
			reg := storage.NoopRegistry{}
			p := agent.NewPeer("local")

			lc := fakeClient{
				status: agent.StatusResponse{
					Deployments: []*agent.Deploy{
						{
							Stage:   agent.Deploy_Completed,
							Archive: &archive1,
							Options: &dopts1,
						},
					},
				},
			}
			fc := fakeClient{
				qinfo: agent.InfoResponse{
					Mode: agent.InfoResponse_Deploying,
					Deploying: &agent.DeployCommand{
						Command: agent.DeployCommand_Done,
						Archive: &archive1,
						Options: &dopts1,
					},
					Deployed: &agent.DeployCommand{
						Command: agent.DeployCommand_Done,
						Archive: &archive2,
						Options: &dopts1,
					},
				},
			}

			fd := fakeDialer{c: fc}
			mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
			dc := deployment.New(
				p,
				noopDeployer{err: nil},
				deployment.CoordinatorOptionStorage(reg),
				deployment.CoordinatorOptionRoot(testingx.TempDir()),
			)
			Expect(errors.Cause(Bootstrap(p, mc, fd, dc))).To(Equal(agentutil.ErrActiveDeployment))
			fdn := fakeDialer{c: fc, local: lc}
			Expect(Bootstrap(p, mc, fdn, dc).Error()).To(Equal("active deploy matches the local deployment, waiting for deployment to complete: deployment in progress"))
		})

		It("should deploy an active deploy from quorum, fallback to quorum vote to pull last successful", func() {
			reg := storage.NoopRegistry{}
			p := agent.NewPeer("local")

			lc := fakeClient{
				status: agent.StatusResponse{
					Deployments: []*agent.Deploy{
						{
							Stage:   agent.Deploy_Completed,
							Archive: &archive2,
							Options: &dopts1,
						},
					},
				},
			}
			fc := fakeClient{
				status: agent.StatusResponse{
					Deployments: []*agent.Deploy{
						{
							Stage:   agent.Deploy_Completed,
							Archive: &archive1,
							Options: &dopts1,
						},
					},
				},
				qinfo: agent.InfoResponse{
					Mode: agent.InfoResponse_Deploying,
					Deploying: &agent.DeployCommand{
						Command: agent.DeployCommand_Done,
						Archive: &archive2,
						Options: &dopts1,
					},
				},
			}

			fd := fakeDialer{c: fc}
			mc := cluster.New(cluster.NewLocal(p), clustering.NewSingleNode("node1", net.ParseIP("127.0.0.1")))
			dc := deployment.New(
				p,
				noopDeployer{err: nil},
				deployment.CoordinatorOptionStorage(reg),
				deployment.CoordinatorOptionRoot(testingx.TempDir()),
			)

			// we (currently) always return no deployments in this case because raft cluster
			// is in a fucked up state. particularly the case of computing quorum for
			// the latest deployment because the raft cluster is missing the last
			// successful deployment but is actively running a deploy is very rare.
			// this way we still attempt to deploy the current active deploy but do not
			// consider the bootstrap complete until the active deploy is complete.
			Expect(errors.Cause(Bootstrap(p, mc, fd, dc))).To(Equal(agentutil.ErrNoDeployments))
			fdn := fakeDialer{c: fc, local: lc}
			Expect(errors.Cause(Bootstrap(p, mc, fdn, dc))).To(Equal(agentutil.ErrNoDeployments))
		})
	})
})

var _ = Describe("UntilSuccess", func() {
	var (
		peer1    = agent.NewPeer("node1")
		archive1 = agent.Archive{
			Peer:         &peer1,
			Ts:           time.Now().Unix(),
			DeploymentID: bw.MustGenerateID(),
		}
		archive2 = agent.Archive{
			Peer:         &peer1,
			Ts:           time.Now().Unix(),
			DeploymentID: bw.MustGenerateID(),
		}
		dopts1 = agent.DeployOptions{
			Timeout:           int64(time.Hour),
			SilenceDeployLogs: true,
		}
	)

	It("should succeed when no errors occur", func() {
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{
					{
						Archive: &archive1,
						Options: &dopts1,
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
		)
		Expect(NewUntilSuccess().Run(p, mc, fd, dc)).To(BeTrue())
	})

	It("should fail when the deployment fails", func() {
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{
					// the latest deploy needs to be a failure to trigger the latest deploy check to fail.
					{
						Stage:   agent.Deploy_Failed,
						Archive: &archive2,
						Options: &dopts1,
					},
					{
						Stage:   agent.Deploy_Completed,
						Archive: &archive1,
						Options: &dopts1,
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
		)
		us := NewUntilSuccess(
			OptionMaxAttempts(10),
			OptionBackoff(backoff.Constant(0)),
		)
		Expect(us.Run(p, mc, fd, dc)).To(BeFalse())
	})
})
