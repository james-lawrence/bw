package agent

import (
	"bytes"
	"fmt"
	"hash"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/grpc/credentials"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/uploads"
	"golang.org/x/net/context"
)

// Coordinator is in charge of coordinating deployments.
type deployer interface {
	// Info obout the deployment coordinator
	// idle, canary, deploying, locked, and the list of recent deployments.
	Info() (agent.AgentInfo, error)
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

func (t noopDeployer) Info() (agent.AgentInfo, error) {
	return agent.AgentInfo{
		Status: agent.AgentInfo_Ready,
	}, nil
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
		s.Deployer = d
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
func NewServer(address net.Addr, creds credentials.TransportCredentials, options ...ServerOption) Server {
	s := Server{
		creds:    creds,
		Address:  address,
		cluster:  noopCluster{},
		Deployer: noopDeployer{},
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
	creds          credentials.TransportCredentials
	Deployer       deployer
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
		chunk        *agent.ArchiveChunk
	)

	if deploymentID, err = bw.GenerateID(); err != nil {
		return err
	}

	if chunk, err = stream.Recv(); err != nil {
		return errors.WithStack(err)
	}

	if dst, err = t.UploadProtocol.NewUpload(deploymentID, chunk.GetMetadata().Bytes); err != nil {
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
				Ts:           time.Now().UTC().Unix(),
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
	if err := t.Deployer.Deploy(archive); err != nil {
		return nil, err
	}

	return &agent.DeployResult{}, nil
}

// Info ...
func (t Server) Info(ctx context.Context, _ *agent.AgentInfoRequest) (*agent.AgentInfo, error) {
	info, err := t.Deployer.Info()
	return &info, err
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
