package agent

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"net"

	"github.com/hashicorp/memberlist"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/uploads"
	"golang.org/x/net/context"
)

// Coordinator is in charge of coordinating deployments.
type deployer interface {
	// Status of the deployment coordinator
	// idle, deploying, locked
	Status() error
	// Deploy trigger a deploy
	Deploy(*agent.Archive) error
}

type cluster interface {
	Members() []*memberlist.Node
}

type noopDeployer struct{}

func (t noopDeployer) Status() error {
	return nil
}

func (t noopDeployer) Deploy(*agent.Archive) error {
	return nil
}

type noopCluster struct{}

func (t noopCluster) Members() []*memberlist.Node {
	return []*memberlist.Node(nil)
}

// ServerOption ...
type ServerOption func(*Server)

// ComposeServerOptions turns a set of server options into a single server option.
func ComposeServerOptions(options ...ServerOption) ServerOption {
	return func(s *Server) {
		for _, opt := range options {
			opt(s)
		}
	}
}

// ServerOptionDeployer ...
func ServerOptionDeployer(d deployer) ServerOption {
	return func(s *Server) {
		s.deployer = d
	}
}

// ServerOptionCluster ...
func ServerOptionCluster(c cluster, k []byte) ServerOption {
	return func(s *Server) {
		s.cluster = c
		s.clusterKey = k
	}
}

// NewServer ...
func NewServer(address net.Addr, options ...ServerOption) Server {
	s := Server{
		Address:  address,
		cluster:  noopCluster{},
		deployer: noopDeployer{},
		UploadProtocol: uploads.ProtocolFunc(
			func(uid []byte, _ uint64) (uploads.Uploader, error) {
				return uploads.NewTempFileUploader()
			},
		),
		messages: agent.MessageBuilder{Node: address},
	}

	for _, opt := range options {
		opt(&s)
	}

	return s
}

// Server ...
type Server struct {
	deployer       deployer
	cluster        cluster
	clusterKey     []byte
	Address        net.Addr
	UploadProtocol uploads.Protocol
	messages       agent.MessageBuilder
}

// Upload ...
func (t Server) Upload(stream agent.Agent_UploadServer) (err error) {
	var (
		deploymentID []byte
		checksum     hash.Hash
		location     string
		dst          Uploader
	)

	if deploymentID, err = bw.GenerateID(); err != nil {
		return err
	}

	if dst, err = t.UploadProtocol.NewUpload(deploymentID, 0); err != nil {
		return err
	}

	for {
		chunk, err := stream.Recv()

		if err == io.EOF {
			if checksum, location, err = dst.Info(); err != nil {
				log.Println("error getting archive info", err)
				return err
			}

			return stream.SendAndClose(&agent.Archive{
				Leader:       t.Address.String(),
				Location:     location,
				Checksum:     checksum.Sum(nil),
				DeploymentID: deploymentID,
			})
		}

		if err != nil {
			log.Println("error receiving chunk", err)
			return err
		}

		if checksum, err = dst.Upload(bytes.NewBuffer(chunk.Data)); err != nil {
			log.Println("error uploading chunk", err)
			return err
		}
	}
}

// Deploy ...
func (t Server) Deploy(ctx context.Context, archive *agent.Archive) (*agent.DeployResult, error) {
	if err := t.deployer.Deploy(archive); err != nil {
		return nil, err
	}

	return &agent.DeployResult{}, nil
}

// Info ...
func (t Server) Info(ctx context.Context, _ *agent.AgentInfoRequest) (*agent.AgentInfo, error) {
	err := t.deployer.Status()
	if status, ok := err.(deployment.Status); ok {
		return &agent.AgentInfo{
			Status: deployment.AgentStateFromStatus(status),
		}, nil
	}

	return nil, err
}

// Credentials ...
func (t Server) Credentials(ctx context.Context, _ *agent.CredentialsRequest) (_zeror *agent.CredentialsResponse, err error) {
	xpeers := t.cluster.Members()
	peers := make([]*agent.Node, 0, len(xpeers))

	for _, p := range xpeers {
		peers = append(peers, &agent.Node{
			Hostname: p.Name,
			Ip:       net.JoinHostPort(p.Addr.String(), fmt.Sprintf("%d", p.Port)),
		})
	}

	return &agent.CredentialsResponse{Secret: t.clusterKey, Peers: peers}, nil
}

// Events ...
func (t Server) Events(archive *agent.Archive, stream agent.Agent_EventsServer) error {
	return errors.New("not implemented")
}
