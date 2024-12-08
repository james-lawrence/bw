package daemons

import (
	"time"

	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/certificatecache"
)

// AgentCertificateCache initializes the certificate cache manager.
func AgentCertificateCache(ctx Context) (err error) {
	// offset the time into the future to refresh the certificate well in advance
	// of the actual expiration. lets encrypt wants 30 days.
	const futureoffset = 31 * 24 * time.Hour

	config := ctx.Config
	client := acme.NewChallenger(ctx.Cluster.Local(), ctx.Cluster, ctx.ACMECache, ctx.Dialer)
	fallback := certificatecache.NewRefreshAgent(config.Credentials.Directory, client)

	return certificatecache.FromConfig(
		config.Credentials.Directory,
		config.Credentials.Mode,
		ctx.ConfigurationFile,
		futureoffset,
		fallback,
	)
}
