package daemons

import (
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/stringsx"
)

// AgentCertificateCache initializes the certificate cache manager.
func AgentCertificateCache(ctx Context) (err error) {
	config := ctx.Config
	client := acme.NewChallenger(ctx.Cluster.Local(), ctx.Cluster, ctx.ACMECache, ctx.Dialer)
	fallback := certificatecache.NewRefreshAgent(config.CredentialsDir, client)

	return certificatecache.FromConfig(
		stringsx.DefaultIfBlank(config.CredentialsDir, config.Credentials.Directory),
		stringsx.DefaultIfBlank(config.CredentialsMode, config.Credentials.Mode),
		ctx.ConfigurationFile,
		fallback,
	)
}
