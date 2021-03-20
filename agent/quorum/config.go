package quorum

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/james-lawrence/bw/agent"
)

// NewConfiguration ...
func NewConfiguration(a Authority, c cluster, d agent.Dialer) Configuration {
	return Configuration{a: a, c: c, d: d}
}

// Configuration ...
type Configuration struct {
	a Authority
	c cluster
	d agent.Dialer
}

// NewConfigurationService ...
func NewConfigurationService(c Configuration) ConfigurationService {
	return ConfigurationService{c: c, m: &sync.Mutex{}}
}

// ConfigurationService ...
type ConfigurationService struct {
	agent.UnimplementedConfigurationServer
	c Configuration
	m *sync.Mutex
}

// TLSUpdate update the TLS.
func (t ConfigurationService) TLSUpdate(ctx context.Context, req *agent.TLSUpdateRequest) (_ *agent.TLSUpdateResponse, err error) {
	t.m.Lock()
	defer t.m.Unlock()

	return &agent.TLSUpdateResponse{}, t.c.a.write(req.Creds)
}

// Encode noop, nothing to persist.
func (t Configuration) Encode(dst io.Writer) (err error) {
	return nil
}

// Decode ...
func (t Configuration) Decode(tctx TranscoderContext, m agent.Message) (err error) {
	var (
		evt agent.TLSEvent
		c   agent.Client
	)

	switch m.GetEvent().(type) {
	case *agent.Message_TLSRequest:
	default:
		return nil
	}

	if m.Peer == nil {
		return errors.New("invalid message, missing peer")
	}

	if evt, err = t.a.read(); err != nil {
		return err
	}

	if c, err = t.d.Dial(*m.Peer); err != nil {
		return err
	}

	req := &agent.TLSUpdateRequest{Creds: &evt}
	if _, err = agent.NewConfigurationClient(c.Conn()).TLSUpdate(context.Background(), req); err != nil {
		return err
	}

	return nil
}
