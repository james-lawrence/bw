package agentcmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/uxterm"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type CmdDaemonDebugRaft struct {
	Location string `arg:"" name:"path" help:"location of the raft log file"`
}

func (t CmdDaemonDebugRaft) Run() (err error) {
	type stats struct {
		barriers       int
		commands       int
		noops          int
		configurations int
		unknown        int
	}

	var (
		lstats stats
	)

	store, err := commandutils.RaftStoreFilepath(t.Location)
	if err != nil {
		return err
	}

	i, err := store.FirstIndex()
	if err != nil {
		return err
	}

	l, err := store.LastIndex()
	if err != nil {
		return err
	}

	for ; i <= l; i++ {
		var (
			current raft.Log
			decoded agent.Message
		)

		if err = store.GetLog(i, &current); err != nil {
			fmt.Println("get log failed", i, err)
			continue
		}

		switch current.Type {
		case raft.LogBarrier:
			lstats.barriers++
			if envx.Boolean(false, bw.EnvLogsVerbose) {
				fmt.Println("barrier invoked", current.Index, current.Term)
			}
			continue
		case raft.LogCommand:
			lstats.commands++
			if err = proto.Unmarshal(current.Data, &decoded); err != nil {
				fmt.Println("decode failed", i, err)
				continue
			}
			fmt.Println("message", prototext.Format(&decoded))
		case raft.LogNoop:
			lstats.noops++
			fmt.Println("noop invoked", current.Index, current.Term)
			continue
		case raft.LogConfiguration:
			lstats.configurations++
			if envx.Boolean(false, bw.EnvLogsVerbose) {
				fmt.Println("log configuration", current.Index, current.Term)
			}
		default:
			lstats.unknown++
			fmt.Println("unexpected log message", current.Type)
			continue
		}
	}

	fmt.Printf("log metrics %#v\n", lstats)
	return nil
}

type CmdDaemonDebugQuorum struct {
	Config
}

func (t *CmdDaemonDebugQuorum) Run(ctx *cmdopts.Global, aconfig *agent.Config) (err error) {
	var (
		ns        notary.Composite
		ss        notary.Signer
		tlsconfig *tls.Config
		conn      *grpc.ClientConn
		d         dialers.ContextDialer
		creds     credentials.TransportCredentials
		quorum    *agent.InfoResponse
		config    = aconfig.Clone()
	)
	defer ctx.Shutdown()

	if config, err = commandutils.LoadAgentConfig(t.Location, config); err != nil {
		return errors.Wrap(err, "unable to load configuration")
	}

	log.Println(spew.Sdump(config.Sanitize()))

	local := config.Peer()

	if creds, err = certificatecache.GRPCGenServer(config); err != nil {
		return err
	}

	if tlsconfig, err = certificatecache.TLSGenServer(config); err != nil {
		return err
	}

	if ns, err = notary.NewFromFile(filepath.Join(config.Root, bw.DirAuthorizations), t.Location); err != nil {
		return err
	}

	if ss, err = commandutils.Generatecredentials(config, ns); err != nil {
		return err
	}

	var di = discovery.ProxyDialer{
		Proxy:  agent.P2PRawAddress(local),
		Signer: ss,
		Dialer: muxer.NewDialer(
			bw.ProtocolProxy,
			tlsx.NewDialer(tlsconfig),
		),
	}

	log.Println("address", agent.RPCAddress(local))
	d = dialers.NewDirect(
		agent.RPCAddress(local),
		dialers.WithMuxer(
			di,
			&net.TCPAddr{IP: net.ParseIP(local.Ip), Port: int(local.P2PPort)},
		),
		grpc.WithTransportCredentials(creds),
	)

	if conn, err = d.DialContext(ctx.Context); err != nil {
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
