package main

import (
	"log"
	"path/filepath"
	"time"

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
	global     *global
	configPath string
	config     agent.Config
}

func (t *agentNotify) configure(parent *kingpin.CmdClause) {
	parent.Flag("agent-config", "configuration file to use").Default(bw.DefaultLocation(bw.DefaultAgentConfig, "")).StringVar(&t.configPath)
	parent.Flag("agent-address", "address of the RPC server to use").PlaceHolder(t.config.RPCBind.String()).TCPVar(&t.config.RPCBind)
	parent.Action(t.exec)
}

func (t *agentNotify) exec(ctx *kingpin.ParseContext) (err error) {
	var (
		client agent.Client
		creds  credentials.TransportCredentials
	)
	defer t.global.shutdown()

	if err = bw.ExpandAndDecodeFile(t.configPath, &t.config); err != nil {
		return err
	}

	log.Println(spew.Sdump(t.config))

	if creds, err = t.config.GRPCCredentials(); err != nil {
		return err
	}

	if client, err = agent.AddressProxyDialQuorum(t.config.RPCBind.String(), grpc.WithTransportCredentials(creds)); err != nil {
		return err
	}

	t.global.cleanup.Add(1)
	go func() {
		defer t.global.cleanup.Done()
		<-t.global.ctx.Done()
		client.Close()
		time.Sleep(5 * time.Second)
	}()

	n, err := notifications.DecodeConfig(filepath.Join(t.config.Root, "notifications.toml"), map[string]notifications.Creator{
		"default": func() notifications.Notifier { return notifications.New() },
		"slack":   func() notifications.Notifier { return slack.New() },
	})
	if err != nil {
		return err
	}

	log.Println(spew.Sdump(n))
	return notifier.New(n...).Start(client)
}
