package agentcmd

import (
	"net"
	"strings"

	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
)

type Config struct {
	Location       string   `name:"agent-config" help:"configuration file to load" default:"${vars_bw_default_agent_configuration_location}"`
	Address        string   `name:"agent-address" alias:"agent-p2p" help:"address for the agent to bind" default:"${vars_bw_default_agent_address}" env:"${env_bw_agent_bind_primary}"`
	P2PAdvertised  string   `name:"agent-address-advertised" alias:"agent-p2p-advertised" help:"ip address to advertise" env:"${env_bw_agent_bind_advertised}"`
	AlternateBinds []string `name:"agent-address-bindings" alias:"agent-p2p-alternates" help:"additional ip/port for the server to bind" placeholder:"127.0.0.1:2000" env:"${env_bw_agent_bind_secondary}"`
}

func (t Config) AfterApply(config *agent.Config) (err error) {
	var (
		addr       *net.TCPAddr
		advertised net.IP
		alternates []*net.TCPAddr
	)

	for _, saddr := range t.AlternateBinds {
		var (
			a *net.TCPAddr
		)

		if a, err = net.ResolveTCPAddr("tcp", saddr); err != nil {
			return err
		}

		alternates = append(alternates, a)
	}

	if addr, err = net.ResolveTCPAddr("tcp", t.Address); err != nil {
		return err
	}

	advertised = net.ParseIP(t.P2PAdvertised)
	if strings.TrimSpace(t.P2PAdvertised) != "" && advertised == nil {
		return errors.Errorf("invalid advertised ip address: %s", t.P2PAdvertised)
	}

	*config = config.Clone(
		agent.ConfigOptionRPC(addr),
		agent.ConfigOptionP2P(addr),
		agent.ConfigOptionAdvertised(advertised),
		agent.ConfigOptionSecondaryBindings(alternates...),
	)

	return nil
}
