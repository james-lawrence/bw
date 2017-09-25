package agent

import (
	"bytes"
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
	// Deploy trigger a deploy
	Deploy(*agent.Archive) error
}

type cluster interface {
	Members() []*memberlist.Node
}

type noopDeployer struct{}

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
func NewServer(info agent.Peer, address net.Addr, creds credentials.TransportCredentials, options ...ServerOption) Server {
	s := Server{
		info:     info,
		creds:    creds,
		Address:  address,
		cluster:  noopCluster{},
		Deployer: noopDeployer{},
		UploadProtocol: uploads.ProtocolFunc(
			func(uid []byte, _ uint64) (uploads.Uploader, error) {
				return uploads.NewTempFileUploader()
			},
		),
	}

	for _, opt := range options {
		opt(&s)
	}

	return s
}

// Server ...
type Server struct {
	Address        net.Addr
	info           agent.Peer
	creds          credentials.TransportCredentials
	Deployer       deployer
	cluster        cluster
	clusterKey     []byte
	UploadProtocol uploads.Protocol
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
				Peer:         &t.info,
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
func (t Server) Info(ctx context.Context, _ *agent.StatusRequest) (*agent.Status, error) {
	return &agent.Status{
		Peer: &t.info,
	}, nil
}

// Credentials ...
func (t Server) Credentials(ctx context.Context, _ *agent.CredentialsRequest) (_zeror *agent.CredentialsResponse, err error) {
	xpeers := t.cluster.Members()
	peers := make([]*agent.Peer, 0, len(xpeers))

	for _, p := range xpeers {
		peers = append(peers, &agent.Peer{
			Status:   agent.Peer_Unknown,
			Name:     p.Name,
			Ip:       p.Addr.String(),
			RPCPort:  t.info.RPCPort,
			SWIMPort: t.info.SWIMPort,
			RaftPort: t.info.RaftPort,
		})
	}

	return &agent.CredentialsResponse{Secret: t.clusterKey, Peers: peers}, nil
}
