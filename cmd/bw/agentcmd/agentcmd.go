package agentcmd

import (
	"net"

	"github.com/james-lawrence/bw/agent"
)

type Config struct {
	Location       string         `name:"agent-config" help:"configuration file to load" default:"${vars_bw_default_agent_configuration_location}"`
	Address        *net.TCPAddr   `name:"agent-address" alias:"agent-p2p" help:"address for the agent to bind" default:"${vars_bw_default_agent_address}" env:"${env_bw_agent_bind_primary}"`
	P2PAdvertised  *net.TCPAddr   `name:"agent-address-advertised" alias:"agent-p2p-advertised" help:"ip address to advertise" env:"${env_bw_agent_bind_advertised}"`
	AlternateBinds []*net.TCPAddr `name:"agent-address-bindings" alias:"agent-p2p-alternates" help:"additional ip/port for the server to bind" placeholder:"127.0.0.1:2000" env:"${env_bw_agent_bind_secondary}"`
}

func (t Config) AfterApply(config *agent.Config) (err error) {
	if t.Address == nil {
		t.Address = config.P2PBind
	}

	*config = config.Clone(
		agent.ConfigOptionP2P(t.Address),
		agent.ConfigOptionAdvertised(t.P2PAdvertised),
		agent.ConfigOptionSecondaryBindings(t.AlternateBinds...),
	).EnsureDefaults()

	return nil
}
