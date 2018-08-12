package main

import (
	"log"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/notifier"
	"github.com/james-lawrence/bw/deployment/notifications"
	"github.com/james-lawrence/bw/deployment/notifications/slack"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type agentNotify struct {
	global      *global
	configPath  string
	nconfigPath string
	config      agent.Config
}

func (t *agentNotify) configure(parent *kingpin.CmdClause) {
	parent.Flag("agent-config", "configuration file to use").Default(bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), "")).StringVar(&t.configPath)
	parent.Flag("agent-address", "address of the RPC server to use").PlaceHolder(t.config.RPCBind.String()).TCPVar(&t.config.RPCBind)
	parent.Flag("notification-config", "name of the notification configuration file in the same directory as the agent config").Default("notifications.toml").StringVar(&t.nconfigPath)
	parent.Action(t.exec)
}

func (t *agentNotify) exec(ctx *kingpin.ParseContext) (err error) {
	var (
		creds credentials.TransportCredentials
	)
	defer t.global.shutdown()

	if err = bw.ExpandAndDecodeFile(t.configPath, &t.config); err != nil {
		return err
	}

	log.Println(spew.Sdump(t.config))

	if creds, err = t.config.GRPCCredentials(); err != nil {
		return err
	}

	n, err := notifications.DecodeConfig(filepath.Join(filepath.Dir(t.configPath), t.nconfigPath), map[string]notifications.Creator{
		"default": func() notifications.Notifier { return notifications.New() },
		"slack":   func() notifications.Notifier { return slack.New() },
	})
	if err != nil {
		return err
	}

	log.Println(spew.Sdump(n))

	notifier.New(n...).Start(t.global.ctx, agent.NewPeer("local"), t.config.Peer(), agent.NewDialer(grpc.WithTransportCredentials(creds)))

	return nil
}
