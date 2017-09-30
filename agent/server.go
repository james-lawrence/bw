package agent

import (
	"bytes"
	"hash"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/grpc/credentials"

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

type noopDeployer struct{}

func (t noopDeployer) Deploy(*agent.Archive) error {
	return nil
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
func ServerOptionCluster(c cluster) ServerOption {
	return func(s *Server) {
		s.cluster = c
	}
}

// NewServer ...
func NewServer(c cluster, address net.Addr, creds credentials.TransportCredentials, options ...ServerOption) Server {
	s := Server{
		creds:    creds,
		cluster:  c,
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
	creds          credentials.TransportCredentials
	Deployer       deployer
	cluster        cluster
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
			tmp := t.cluster.Local()
			return stream.SendAndClose(&agent.Archive{
				Peer:         &tmp,
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
	tmp := t.cluster.Local()
	return &agent.Status{
		Peer: &tmp,
	}, nil
}

// Quorum ...
func (t Server) Quorum(ctx context.Context, _ *agent.DetailsRequest) (_zeror *agent.Details, err error) {
	details := t.cluster.Details()
	return &details, nil
}
