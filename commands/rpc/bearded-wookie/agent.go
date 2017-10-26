package main

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agent/proxy"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/peering"
	"bitbucket.org/jatone/bearded-wookie/clustering/raftutil"
	"bitbucket.org/jatone/bearded-wookie/uploads"
	"bitbucket.org/jatone/bearded-wookie/x/stringsx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type agentCmd struct {
	*global
	config     agent.Config
	configFile string
	server     *grpc.Server
	listener   net.Listener
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(
		parent,
		clusterCmdOptionName(t.config.Name),
		clusterCmdOptionBind(t.config.SWIMBind),
		clusterCmdOptionRaftBind(t.config.RaftBind),
	)

	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("agent-bind", "address for the RPC server to bind to").PlaceHolder(t.config.RPCBind.String()).TCPVar(&t.config.RPCBind)
	parent.Flag("agent-config", "file containing the agent configuration").
		Default(bw.DefaultLocation(bw.DefaultAgentConfig, "")).StringVar(&t.configFile)

	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())
	(&dummy{agentCmd: t}).configure(parent.Command("dummy", "dummy deployment"))
}

func (t *agentCmd) bind(aoptions func(*agentutil.Dispatcher, agent.Peer, agent.Config) agent.ServerOption) error {
	var (
		err    error
		c      clustering.Cluster
		creds  *tls.Config
		secret []byte
		p      raftutil.Protocol
		upload uploads.Protocol
	)
	log.SetPrefix("[AGENT] ")

	if err = bw.ExpandAndDecodeFile(t.configFile, &t.config); err != nil {
		return err
	}

	log.Printf("configuration: %#v\n", t.config)
	if upload, err = t.config.Storage.Protocol(); err != nil {
		return err
	}

	if t.listener, err = net.Listen(t.config.RPCBind.Network(), t.config.RPCBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", t.config.RPCBind)
	}

	if secret, err = t.config.TLSConfig.Hash(); err != nil {
		return err
	}

	if creds, err = t.config.TLSConfig.BuildServer(); err != nil {
		return err
	}

	if p, err = t.global.cluster.Raft(t.global.ctx, t.config); err != nil {
		return err
	}

	local := cluster.NewLocal(agent.Peer{
		Ip:       t.global.systemIP.String(),
		Name:     stringsx.DefaultIfBlank(t.config.Name, t.listener.Addr().String()),
		RPCPort:  uint32(t.config.RPCBind.Port),
		SWIMPort: uint32(t.config.SWIMBind.Port),
		RaftPort: uint32(t.config.RaftBind.Port),
	})

	tlscreds := credentials.NewTLS(creds)
	keepalive := grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    10 * time.Second,
		Timeout: 3 * time.Second,
	})
	t.server = grpc.NewServer(grpc.Creds(tlscreds), keepalive)
	options := []clustering.Option{
		clustering.OptionNodeID(local.Peer.Name),
		clustering.OptionDelegate(local),
		clustering.OptionLogOutput(os.Stderr),
		clustering.OptionSecret(secret),
		clustering.OptionEventDelegate(&p),
		clustering.OptionAliveDelegate(cluster.AliveDefault{}),
	}

	fssnapshot := peering.File{
		Path: filepath.Join(t.config.Root, "cluster.snapshot"),
	}

	if c, err = t.global.cluster.Join(fssnapshot, options...); err != nil {
		return errors.Wrap(err, "failed to join cluster")
	}

	t.global.cluster.Snapshot(
		c,
		fssnapshot,
		clustering.SnapshotOptionFrequency(t.config.Cluster.SnapshotFrequency),
		clustering.SnapshotOptionContext(t.global.ctx),
	)

	lq := agent.NewQuorumFSM()
	cx := cluster.New(local, c)

	dispatcher := agentutil.NewDispatcher(cx, grpc.WithTransportCredentials(tlscreds))
	quorum := agent.NewQuorum(
		lq,
		cx,
		proxy.NewProxy(cx, dispatcher),
		agent.QuorumOptionUpload(upload),
		agent.QuorumOptionCredentials(tlscreds),
	)
	server := agent.NewServer(
		cx,
		aoptions(dispatcher, local.Peer, t.config),
	)

	agent.RegisterServer(
		t.server,
		server,
	)

	agent.RegisterQuorum(t.server, quorum)

	t.runServer(c)

	go p.Overlay(
		c,
		raftutil.ProtocolOptionStateMachine(func() raft.FSM {
			return lq
		}),
		raftutil.ProtocolOptionClusterObserver(
			agentutil.NewBootstrapper(cx, tlscreds),
		),
		raftutil.ProtocolOptionObservers(
			raft.NewObserver(quorum.Events, true, func(o *raft.Observation) bool {
				switch o.Data.(type) {
				case raft.RaftState:
					return true
				default:
					return false
				}
			}),
			// TODO: remove the ProtocolOptionClusterObserver and replace it with a cluster monitor.
			// then wire bootstrap and other systems up to the cluster monitor.
			// raft.NewObserver(clustermonitor.Events, true, func(o *raft.Observation) bool {
			// 	_, ok := o.Data.(raft.LeaderObservation)
			// 	return ok
			// }),
		),
	)

	return nil
}

func (t *agentCmd) runServer(c clustering.Cluster) {
	go t.server.Serve(t.listener)
	t.global.cleanup.Add(1)
	go func() {
		defer t.global.cleanup.Done()
		<-t.global.ctx.Done()

		log.Println("left cluster", c.Shutdown())
		log.Println("agent shutdown", t.listener.Close())
	}()
}
