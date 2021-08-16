package main

import (
	"crypto/tls"
	"log"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/notifier"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment/notifications"
	"github.com/james-lawrence/bw/deployment/notifications/native"
	"github.com/james-lawrence/bw/deployment/notifications/slack"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/notary"
	"google.golang.org/grpc"
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
		ns        notary.Composite
		ss        notary.Signer
		tlsconfig *tls.Config
	)
	defer t.global.shutdown()

	if t.config, err = commandutils.LoadAgentConfig(t.configPath, t.config); err != nil {
		return err
	}

	log.Println(spew.Sdump(t.config))

	if tlsconfig, err = daemons.TLSGenServer(t.config, tlsx.OptionNoClientCert); err != nil {
		return err
	}

	if ns, err = notary.NewFromFile(filepath.Join(t.config.Root, bw.DirAuthorizations), t.configPath); err != nil {
		return err
	}

	if ss, err = generatecredentials(t.config, ns); err != nil {
		return err
	}

	n, err := notifications.DecodeConfig(filepath.Join(filepath.Dir(t.configPath), t.nconfigPath), map[string]notifications.Creator{
		"default": func() notifications.Notifier { return notifications.New() },
		"desktop": func() notifications.Notifier { return native.New() },
		"slack":   func() notifications.Notifier { return slack.New() },
	})
	if err != nil {
		return err
	}

	log.Println(spew.Sdump(n))

	d, err := dialers.DefaultDialer(agent.P2PRawAddress(t.config.Peer()), tlsx.NewDialer(tlsconfig), grpc.WithPerRPCCredentials(ss))
	if err != nil {
		return err
	}
	dd := dialers.NewProxy(dialers.NewDirect(agent.RPCAddress(t.config.Peer()), d.Defaults(grpc.WithPerRPCCredentials(ss))...))

	notifier.New(n...).Start(t.global.ctx, agent.NewPeer("local"), t.config.Peer(), dd)

	return nil
}
