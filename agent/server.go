package agent

import (
	"bytes"
	"errors"
	"hash"
	"io"
	"log"
	"net"

	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
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

type noopDeployer struct{}

func (t noopDeployer) Status() error {
	return nil
}

func (t noopDeployer) Deploy(*agent.Archive) error {
	return nil
}

// ServerOption ...
type ServerOption func(*Server)

// ServerOptionDeployer ...
func ServerOptionDeployer(d deployer) ServerOption {
	return func(s *Server) {
		s.deployer = d
	}
}

// NewServer ...
func NewServer(address net.Addr, options ...ServerOption) Server {
	s := Server{
		Address:     address,
		deployer:    noopDeployer{},
		NewUploader: func() (Uploader, error) { return NewFileUploader() },
		messages:    agent.MessageBuilder{Node: address},
	}

	for _, opt := range options {
		opt(&s)
	}

	return s
}

// Server ...
type Server struct {
	deployer    deployer
	Address     net.Addr
	NewUploader func() (Uploader, error)
	messages    agent.MessageBuilder
}

// Upload ...
func (t Server) Upload(stream agent.Agent_UploadServer) (err error) {
	var (
		deploymentID []byte
		checksum     hash.Hash
		location     string
		dst          Uploader
	)

	if deploymentID, err = GenerateID(); err != nil {
		return err
	}

	if dst, err = t.NewUploader(); err != nil {
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

// Events ...
func (t Server) Events(archive *agent.Archive, stream agent.Agent_EventsServer) error {
	return errors.New("not implemented")
}
