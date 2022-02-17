package agentcmd

import (
	"crypto/tls"
	"log"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/notifier"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/cmd/bwc/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/deployment/notifications"
	"github.com/james-lawrence/bw/deployment/notifications/native"
	"github.com/james-lawrence/bw/deployment/notifications/slack"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/notary"
	"google.golang.org/grpc"
)

type Notify struct {
	Config
	Notifications string `name:"notification-config" help:"name of the notification configuration file in the same directory as the agent config" default:"notifications.toml"`
}

func (t *Notify) Run(ctx *cmdopts.Global, configctx *agent.Config) (err error) {
	var (
		ns        notary.Composite
		ss        notary.Signer
		tlsconfig *tls.Config
		config    = configctx.Clone()
	)
	defer ctx.Shutdown()

	if config, err = commandutils.LoadAgentConfig(t.Location, config); err != nil {
		return err
	}

	log.Println(spew.Sdump(config))

	if tlsconfig, err = certificatecache.TLSGenServer(config, tlsx.OptionNoClientCert); err != nil {
		return err
	}

	if ns, err = notary.NewFromFile(filepath.Join(config.Root, bw.DirAuthorizations), t.Location); err != nil {
		return err
	}

	if ss, err = commandutils.Generatecredentials(config, ns); err != nil {
		return err
	}

	n, err := notifications.DecodeConfig(filepath.Join(filepath.Dir(t.Location), t.Notifications), map[string]notifications.Creator{
		"default": func() notifications.Notifier { return notifications.New() },
		"desktop": func() notifications.Notifier { return native.New() },
		"slack":   func() notifications.Notifier { return slack.New() },
	})
	if err != nil {
		return err
	}

	log.Println(spew.Sdump(n))

	d, err := dialers.DefaultDialer(agent.P2PRawAddress(config.Peer()), tlsx.NewDialer(tlsconfig), grpc.WithPerRPCCredentials(ss))
	if err != nil {
		return err
	}
	dd := dialers.NewProxy(dialers.NewDirect(agent.RPCAddress(config.Peer()), d.Defaults()...))

	notifier.New(n...).Start(ctx.Context, agent.NewPeer("local"), config.Peer(), dd)

	return nil
}
