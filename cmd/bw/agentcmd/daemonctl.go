package agentcmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/gofrs/uuid"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/debug"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/operations"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/rendezvous"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/systemx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/uxterm"
	"github.com/logrusorgru/aurora"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type CmdControl struct {
	Restart    CmdControlRestart       `cmd:"" help:"restart all the nodes within the cluster"`
	Quorum     CmdControlQuorum        `cmd:"" help:"print information about the quorum members of the cluster"`
	Stacktrace CmdControlStacktrace    `cmd:"" help:"print stack trace from each node"`
	CPU        CmdControlProfileCPU    `cmd:"" help:"run a cpu profile against agents"`
	Memory     CmdControlProfileMemory `cmd:"" help:"run a memory profile against agents"`
	Heap       CmdControlProfileHeap   `cmd:"" help:"run a heap profile against agents"`
}

type controlConnection struct {
	cmdopts.BeardedWookieEnv
	Names    []*regexp.Regexp `name:"name" help:"regex to match names against"`
	IPs      []net.IP         `name:"ip" help:"match against the provided IP addresses"`
	Canary   bool             `name:"canary" help:"deploy to the canary server" default:"false"`
	Insecure bool             `name:"insecure" help:"disable tls verification"`
}

func (t controlConnection) connect() (d dialers.Defaults, c clustering.Rendezvous, err error) {
	var (
		config agent.ConfigClient
		ss     notary.Signer
	)

	if config, err = commandutils.LoadConfiguration(t.Environment, agent.CCOptionInsecure(t.Insecure)); err != nil {
		return d, c, err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return d, c, err
	}

	if d, c, err = daemons.Connect(config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return d, c, err
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("connected to cluster")
	}

	return d, c, nil
}

func (t controlConnection) filters(additional ...deployment.Filter) []deployment.Filter {
	filters := make([]deployment.Filter, 0, 1+len(t.Names)+len(t.IPs))
	for _, n := range t.Names {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.IPs {
		filters = append(filters, deployment.IP(n))
	}

	for _, a := range additional {
		if a == nil {
			continue
		}

		filters = append(filters, a)
	}

	if len(filters) == 0 {
		filters = append(filters, deployment.AlwaysMatch)
	}

	return filters
}

func (t controlConnection) canary(c clustering.Rendezvous) deployment.Filter {
	if t.Canary {
		return deployment.IP(c.Get(rendezvous.Auto()).Addr)
	}

	return nil
}

func (t controlConnection) filtered(c clustering.Rendezvous) clustering.Rendezvous {
	filters := t.filters(t.canary(c))
	return clustering.NewCached(func(ctx context.Context) clustering.Rendezvous {
		peers := agent.NodesToPeers(c.Members()...)
		return clustering.NewStatic(agent.PeersToNodes(
			deployment.ApplyFilter(deployment.Or(filters...), peers...)...,
		)...)
	})
}

type CmdControlRestart struct {
	controlConnection
	Enabled bool `name:"force" help:"must be specified in order for the command to actual be sent" default:"false"`
}

func (t CmdControlRestart) Run(ctx *cmdopts.Global) error {
	return t.shutdown(ctx.Context, deployment.Or(t.filters()...))
}

func (t CmdControlRestart) shutdown(ctx context.Context, filter deployment.Filter) (err error) {
	var (
		d dialers.Defaults
		c clustering.Rendezvous
	)

	local := &agent.Peer{
		Name: bw.MustGenerateID().String(),
		Ip:   systemx.HostnameOrLocalhost(),
	}

	if d, c, err = t.connect(); err != nil {
		return err
	}

	cx := cluster.New(local, c)

	peers := agentutil.PeerSet(deployment.ApplyFilter(filter, cx.Peers()...))
	if !t.Enabled {
		log.Println("force not specified, not executing for the following agents:")
		for _, p := range peers.Peers() {
			log.Println(p.Name, p.Ip)
		}
		return nil
	}

	return agentutil.Shutdown(ctx, peers, d)
}

type CmdControlQuorum struct {
	controlConnection
}

func (t CmdControlQuorum) Run(ctx *cmdopts.Global) (err error) {
	var (
		conn   *grpc.ClientConn
		d      dialers.Defaults
		c      clustering.Rendezvous
		quorum *agent.InfoResponse
	)

	if d, c, err = t.connect(); err != nil {
		return err
	}

	if conn, err = dialers.NewQuorum(c).Dial(d.Defaults()...); err != nil {
		return err
	}

	if quorum, err = agent.NewQuorumClient(conn).Info(ctx.Context, &agent.InfoRequest{}); err != nil {
		return err
	}

	if err = uxterm.PrintQuorum(quorum); err != nil {
		return err
	}

	return nil
}

type CmdControlStacktrace struct {
	controlConnection
}

func (t CmdControlStacktrace) Run(ctx *cmdopts.Global) (err error) {
	var (
		d  dialers.Defaults
		c  clustering.Rendezvous
		au = aurora.NewAurora(true)
	)

	if d, c, err = t.connect(); err != nil {
		return err
	}

	err = operations.New(ctx.Context, operations.Fn(func(ctx context.Context, p *agent.Peer, conn grpc.ClientConnInterface) error {
		var (
			stack *debug.StacktraceResponse
		)

		if stack, err = debug.NewDebugClient(conn).Stacktrace(ctx, &debug.StacktraceRequest{}); err != nil {
			log.Println(au.Red(fmt.Sprint("BEGIN STACKTRACE UNAVAILABLE:", uxterm.PeerString(p))))
			log.Println(err)
			log.Println(au.Red(fmt.Sprint("CEASE STACKTRACE UNAVAILABLE:", uxterm.PeerString(p))))
			return nil
		}

		log.Println(au.Yellow(fmt.Sprint("BEGIN STACKTRACE:", uxterm.PeerString(p))))
		log.Println(string(stack.Trace))
		log.Println(au.Yellow(fmt.Sprint("CEASE STACKTRACE:", uxterm.PeerString(p))))

		return nil
	}))(t.filtered(c), d)

	return nil
}

type CmdControlProfileCPU struct {
	CmdControlProfile
}

func (t CmdControlProfileCPU) Run(ctx *cmdopts.Global) (err error) {
	return t.Do(ctx, func(c debug.DebugClient, req *debug.ProfileRequest) error {
		_, err := c.CPU(ctx.Context, req)
		return err
	})
}

type CmdControlProfileMemory struct {
	CmdControlProfile
}

func (t CmdControlProfileMemory) Run(ctx *cmdopts.Global) (err error) {
	return t.Do(ctx, func(c debug.DebugClient, req *debug.ProfileRequest) error {
		_, err := c.Memory(ctx.Context, req)
		return err
	})
}

type CmdControlProfileHeap struct {
	CmdControlProfile
}

func (t CmdControlProfileHeap) Run(ctx *cmdopts.Global) (err error) {
	return t.Do(ctx, func(c debug.DebugClient, req *debug.ProfileRequest) error {
		_, err := c.Heap(ctx.Context, req)
		return err
	})
}

type CmdControlProfile struct {
	Duration time.Duration `name:"duration" help:"how long to record data" default:"15s"`
	controlConnection
}

func (t CmdControlProfile) Do(ctx *cmdopts.Global, op func(debug.DebugClient, *debug.ProfileRequest) error) (err error) {
	var (
		d  dialers.Defaults
		c  clustering.Rendezvous
		au = aurora.NewAurora(true)
	)

	if d, c, err = t.connect(); err != nil {
		return err
	}

	return operations.New(ctx.Context, operations.Fn(func(ctx context.Context, p *agent.Peer, conn grpc.ClientConnInterface) error {
		client := debug.NewDebugClient(conn)
		id := uuid.Must(uuid.NewV4())

		if _, err = client.Cancel(ctx, &debug.CancelRequest{}); err != nil {
			log.Println(au.Red(fmt.Sprint("BEGIN PROFILE UNAVAILABLE:", uxterm.PeerString(p))))
			log.Println(err)
			log.Println(au.Red(fmt.Sprint("CEASE PROFILE UNAVAILABLE:", uxterm.PeerString(p))))
			return nil
		}

		if err = op(client, &debug.ProfileRequest{
			Id:       id.String(),
			Duration: int64(t.Duration),
		}); err != nil {
			log.Println(au.Red(fmt.Sprint("BEGIN PROFILE UNAVAILABLE:", uxterm.PeerString(p))))
			log.Println(err)
			log.Println(au.Red(fmt.Sprint("CEASE PROFILE UNAVAILABLE:", uxterm.PeerString(p))))
			return nil
		}

		err = grpcx.Retry(func() (err error) {
			var (
				profile *debug.DownloadResponse
			)

			// log.Println("attempting to download profile")

			if profile, err = client.Download(ctx, &debug.DownloadRequest{
				Id: id.String(),
			}); err != nil {
				// errorsx.MaybeLog(err)
				return err
			}

			if _, err = io.Copy(os.Stdout, bytes.NewBuffer(profile.Profile)); err != nil {
				return err
			}

			return nil
		}, codes.Unavailable)

		return err
	}))(t.filtered(c), d)
}
