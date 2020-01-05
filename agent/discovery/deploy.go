package discovery

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
)

type proxydialer interface {
	dialers.Dialer
	dialers.Defaults
}

// NewDeploy deploy client.
func NewDeploy(discovery string, d proxydialer) (_ agent.DeployClient, err error) {
	// deprecated code path.
	if len(discovery) == 0 {
		return agent.MaybeClient(d.Dial())
	}

	return agent.MaybeDeployConn(dialers.NewDirect(discovery).Dial(d.Defaults()...))
}
