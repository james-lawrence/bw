package agent

import (
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/uploads"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"
)

// NewConfigClient ...
func NewConfigClient() ConfigClient {
	return ConfigClient{
		Address:   systemx.HostnameOrLocalhost(),
		TLSConfig: NewTLSClient(DefaultTLSCredentialsRoot),
	}
}

// ConfigClient ...
type ConfigClient struct {
	Address string
	TLSConfig
}

// NewConfig creates a default configuration.
func NewConfig() Config {
	return Config{
		Root:      filepath.Join("/", "var", "cache", bw.DefaultDir),
		TLSConfig: NewTLSAgent(DefaultTLSCredentialsRoot, ""),
		Storage:   uploads.Config{Backend: "local"},
	}
}

// Config - configuration for the agent.
type Config struct {
	Root      string // root directory to store long term data.
	Storage   uploads.Config
	TLSConfig TLSConfig
}
