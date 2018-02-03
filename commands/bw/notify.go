package main

import (
	"log"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
)

type agentNotify struct {
	global     *global
	configPath string
	config     agent.Config
}

func (t *agentNotify) configure(parent *kingpin.CmdClause) {
	parent.Flag("agent-unix-socket", "unix socket address of the agent").StringVar(&t.config.UnixDomainSocketPath)
	parent.Arg("config", "configuration file to use").Default(bw.DefaultLocation(bw.DefaultAgentConfig, "")).StringVar(&t.configPath)
	parent.Action(t.exec)
}

func (t *agentNotify) exec(ctx *kingpin.ParseContext) (err error) {

	defer t.global.shutdown()

	if err = bw.ExpandAndDecodeFile(t.configPath, &t.config); err != nil {
		return err
	}

	log.Println(spew.Sdump(t.config))

	if t.config.UnixDomainSocketPath == "" {
		return errors.New("unix dispatch is blank, cannot connect to an agent with a disabled socket")
	}

	return nil
}
