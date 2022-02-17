package agentcmd

import (
	"net"

	"github.com/james-lawrence/bw/agent"
)

type Config struct {
	Location string `name:"agent-config" help:"configuration file to load" default:"${vars.bw.default.agent.configuration.location}"`
	Address  string `name:"agent-address" help:"address for the agent to bind"  placeholder:"${vars.bw.placeholder.agent.address}"`
}

func (t Config) BeforeApply(config *agent.Config) (err error) {
	var (
		addr *net.TCPAddr
	)

	if addr, err = net.ResolveTCPAddr("tcp", t.Address); err != nil {
		return err
	}

	*config = config.Clone(
		agent.ConfigOptionRPC(addr),
	)

	return nil
}
