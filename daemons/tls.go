package daemons

import (
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/certificatecache"
)

// AgentCertificateCache initializes the certificate cache manager.
func AgentCertificateCache(ctx Context) (err error) {
	config := ctx.Config
	client := acme.NewChallenger(ctx.Cluster.Local(), ctx.Cluster, ctx.ACMECache, ctx.Dialer)
	fallback := certificatecache.NewRefreshAgent(config.Credentials.Directory, client)

	return certificatecache.FromConfig(
		config.Credentials.Directory,
		config.Credentials.Mode,
		ctx.ConfigurationFile,
		fallback,
	)
}
