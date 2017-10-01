package agent

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"bitbucket.org/jatone/bearded-wookie"
	clusterx "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/peering"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/uploads"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"
)

// NewConfigClient ...
func NewConfigClient() ConfigClient {
	return ConfigClient{
		Address:   systemx.HostnameOrLocalhost(),
		TLSConfig: NewTLSClient(DefaultTLSCredentialsRoot),
	}
}

// ConfigClient ...
type ConfigClient struct {
	Address string
	TLSConfig
}

// Connect to the address in the config client.
func (t ConfigClient) Connect(options ...ConnectOption) (creds credentials.TransportCredentials, client Conn, c clustering.Cluster, err error) {
	var (
		details agent.ConnectInfo
	)

	conn := newConnect(options...)

	if creds, client, details, err = t.connect(); err != nil {
		return creds, client, c, err
	}

	if c, err = t.cluster(details, conn.clustering.Options, conn.clustering.Bootstrap); err != nil {
		return creds, client, c, err
	}

	return creds, client, c, nil
}

// ConnectLeader ...
func (t ConfigClient) ConnectLeader(options ...ConnectOption) (creds credentials.TransportCredentials, client Conn, c clustering.Cluster, err error) {
	var (
		details agent.ConnectInfo
		success bool
		tmp     Conn
	)
	conn := newConnect(options...)

	if creds, tmp, details, err = t.connect(); err != nil {
		return creds, client, c, err
	}
	defer tmp.Close()

	for _, p := range details.Quorum {
		if client, err = Dial(clusterx.RPCAddress(*p), grpc.WithTransportCredentials(creds)); err != nil {
			log.Println("failed to connect to peer", p.Name, p.Ip, err)
			continue
		}
		success = true
	}

	if !success {
		return creds, client, c, errors.New("failed to connect to a member of the quorum")
	}

	if c, err = t.cluster(details, conn.clustering.Options, conn.clustering.Bootstrap); err != nil {
		return creds, client, c, err
	}

	return creds, client, c, nil
}

func (t ConfigClient) connect() (creds credentials.TransportCredentials, client Conn, details agent.ConnectInfo, err error) {
	if creds, err = t.creds(); err != nil {
		return creds, client, details, err
	}

	if client, err = Dial(t.Address, grpc.WithTransportCredentials(creds)); err != nil {
		return creds, client, details, err
	}

	if details, err = client.Connect(); err != nil {
		return creds, client, details, err
	}

	return creds, client, details, nil
}

func (t ConfigClient) cluster(details agent.ConnectInfo, copts []clustering.Option, bopts []clustering.BootstrapOption) (c clustering.Cluster, err error) {
	copts = append([]clustering.Option{
		clustering.OptionBindPort(0),
		clustering.OptionAliveDelegate(clusterx.AliveDefault{}),
		clustering.OptionLogOutput(os.Stderr),
		clustering.OptionSecret(details.Secret),
	}, copts...)

	peers := make([]string, 0, len(details.Quorum))
	for _, p := range details.Quorum {
		peers = append(peers, clusterx.SWIMAddress(*p))
	}

	if c, err = clustering.NewOptions(copts...).NewCluster(); err != nil {
		return c, errors.Wrap(err, "failed to join cluster")
	}

	bopts = append([]clustering.BootstrapOption{
		clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(1)),
		clustering.BootstrapOptionAllowRetry(clustering.UnlimitedAttempts),
		clustering.BootstrapOptionPeeringStrategies(
			peering.Closure(func() ([]string, error) {
				return peers, nil
			}),
		),
	}, bopts...)

	if err = clustering.Bootstrap(c, bopts...); err != nil {
		return c, errors.Wrap(err, "failed to connect to cluster")
	}

	return c, nil
}

func (t ConfigClient) creds() (credentials.TransportCredentials, error) {
	var (
		err  error
		conf *tls.Config
	)

	if conf, err = t.TLSConfig.BuildClient(); err != nil {
		return nil, err
	}

	return credentials.NewTLS(conf), nil
}

// NewConfig creates a default configuration.
func NewConfig(options ...ConfigOption) Config {
	c := Config{
		Name:      systemx.HostnameOrLocalhost(),
		Root:      filepath.Join("/", "var", "cache", bw.DefaultDir),
		KeepN:     3,
		TLSConfig: NewTLSAgent(DefaultTLSCredentialsRoot, ""),
		Storage:   uploads.Config{Backend: "local"},
		Cluster: clusteringConfig{
			SnapshotFrequency: time.Hour,
		},
	}

	for _, opt := range options {
		opt(&c)
	}

	return c
}

type clusteringConfig struct {
	SnapshotFrequency time.Duration
}

// ConfigOption - for overriding configurations.
type ConfigOption func(*Config)

// ConfigOptionCompose allow grouping together configuration options to be applied simultaneously.
func ConfigOptionCompose(options ...ConfigOption) ConfigOption {
	return func(c *Config) {
		for _, opt := range options {
			opt(c)
		}
	}
}

// ConfigOptionDefaultBind default connection bindings.
func ConfigOptionDefaultBind(ip net.IP) ConfigOption {
	return ConfigOptionCompose(
		ConfigOptionRPC(&net.TCPAddr{
			IP:   ip,
			Port: 2000,
		}),
		ConfigOptionSWIM(&net.TCPAddr{
			IP:   ip,
			Port: 2001,
		}),
		ConfigOptionRaft(&net.TCPAddr{
			IP:   ip,
			Port: 2002,
		}),
	)
}

// ConfigOptionRPC sets the RPC address to bind.
func ConfigOptionRPC(p *net.TCPAddr) ConfigOption {
	return func(c *Config) {
		c.RPCBind = p
	}
}

// ConfigOptionSWIM sets the SWIM address to bind.
func ConfigOptionSWIM(p *net.TCPAddr) ConfigOption {
	return func(c *Config) {
		c.SWIMBind = p
	}
}

// ConfigOptionRaft sets the Raft address to bind.
func ConfigOptionRaft(p *net.TCPAddr) ConfigOption {
	return func(c *Config) {
		c.RaftBind = p
	}
}

// Config - configuration for the agent.
type Config struct {
	Name      string
	Root      string // root directory to store long term data.
	KeepN     int    `yaml:"keepN"`
	RPCBind   *net.TCPAddr
	RaftBind  *net.TCPAddr
	SWIMBind  *net.TCPAddr
	Storage   uploads.Config
	TLSConfig TLSConfig
	Cluster   clusteringConfig
}

// Peer - builds the agent.Peer information from the configuration. by default
// a peer starts in the unknown state.
func (t Config) Peer() agent.Peer {
	// TODO: have a separate advertise address for the IP field.
	return agent.Peer{
		Status:   agent.Peer_Ready,
		Name:     t.Name,
		Ip:       t.RPCBind.IP.String(),
		RPCPort:  uint32(t.RPCBind.Port),
		RaftPort: uint32(t.RaftBind.Port),
		SWIMPort: uint32(t.SWIMBind.Port),
	}
}
