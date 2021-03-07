package bootstrap_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	. "github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/internal/x/testingx"
	"google.golang.org/grpc"
)

var _ = Describe("Local", func() {
	var (
		peer1    = agent.NewPeer("node1")
		archive1 = agent.Archive{
			Peer:         peer1,
			Ts:           time.Now().Unix(),
			DeploymentID: bw.MustGenerateID(),
		}
		dopts1 = agent.DeployOptions{
			Timeout:           int64(time.Hour),
			SilenceDeployLogs: true,
		}
	)

	It("should succeed when no errors occur", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		p := agent.NewPeer("local")
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
		}

		fd := fakeDialer{local: fc}
		Expect(Run(context.Background(), SocketQuorum(c), NewLocal(p, fd))).To(Succeed())
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithInsecure())
		Expect(err).To(Succeed())
	})

	It("should return no deployments error when no deployments exist", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		p := agent.NewPeer("local")
		fc := fakeClient{
			status: agent.StatusResponse{
				Deployments: []*agent.Deploy{},
			},
		}

		fd := fakeDialer{c: fc}
		Expect(Run(context.Background(), SocketQuorum(c), NewLocal(p, fd))).To(Succeed())
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithInsecure())
		Expect(err).To(Equal(agentutil.ErrNoDeployments))
	})

	It("should error out when an error occurrs", func() {
		c := agent.Config{
			Root: testingx.TempDir(),
		}
		p := agent.NewPeer("local")
		fc := fakeClient{
			errResult: errors.New("boom"),
		}

		fd := fakeDialer{c: fc}
		Expect(Run(context.Background(), SocketQuorum(c), NewLocal(p, fd))).To(Succeed())
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithInsecure())
		Expect(err.Error()).To(Equal("no deployments found"))
	})
})
