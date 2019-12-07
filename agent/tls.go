package agent

import (
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/stringsx"
	"github.com/james-lawrence/bw/internal/x/systemx"
)

// ConfigClientTLS ...
func ConfigClientTLS(credentials string) ConfigClientOption {
	return func(c *ConfigClient) {
		c.CredentialsDir = bw.DefaultUserDirLocation(credentials)
		c.CA = bw.DefaultUserDirLocation(filepath.Join(credentials, bw.DefaultTLSCertCA))
		c.ServerName = stringsx.DefaultIfBlank(c.ServerName, systemx.HostnameOrLocalhost())
	}
}

// NewTLSAgent ...
func newTLSAgent(environment, override string) ConfigOption {
	return func(c *Config) {
		c.CA = bw.DefaultLocation(filepath.Join(environment, bw.DefaultTLSCertCA), override)
		c.CredentialsDir = bw.DefaultLocation(environment, override)
		c.ServerName = stringsx.DefaultIfBlank(c.ServerName, systemx.HostnameOrLocalhost())
	}
}
