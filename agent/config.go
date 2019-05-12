package agent

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"io/ioutil"
	"math"
	"net"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/internal/x/systemx"
)

// ConfigClientOption options for the client configuration.
type ConfigClientOption func(*ConfigClient)

// CCOptionTLSConfig tls set to load for the client configuration.
func CCOptionTLSConfig(name string) ConfigClientOption {
	return ConfigClientTLS(name)
}

// CCOptionAddress set address for the configuration.
func CCOptionAddress(s string) ConfigClientOption {
	return func(c *ConfigClient) {
		c.Address = net.JoinHostPort(s, strconv.Itoa(DefaultPortRPC))
		c.ServerName = s
	}
}

// CCOptionDeployDataDir set the deployment configuration directory for the configuration.
func CCOptionDeployDataDir(s string) ConfigClientOption {
	return func(c *ConfigClient) {
		c.DeployDataDir = s
	}
}

// CCOptionConcurrency set the deployment configuration directory for the configuration.
func CCOptionConcurrency(d float64) ConfigClientOption {
	return func(c *ConfigClient) {
		c.Concurrency = d
	}
}

// CCOptionEnvironment set the environment string for the configuration.
func CCOptionEnvironment(s string) ConfigClientOption {
	return func(c *ConfigClient) {
		c.Environment = s
	}
}

// NewConfigClient ...
func NewConfigClient(template ConfigClient, options ...ConfigClientOption) ConfigClient {
	for _, opt := range options {
		opt(&template)
	}

	return template
}

// DefaultConfigClient creates a default client configuration.
func DefaultConfigClient(options ...ConfigClientOption) ConfigClient {
	config := ConfigClient{
		DeployTimeout: bw.DefaultDeployTimeout,
		Address:       systemx.HostnameOrLocalhost(),
		DeployDataDir: bw.LocateDeployspace(bw.DefaultDeployspaceDir),
	}

	ConfigClientTLS(certificatecache.DefaultTLSCredentialsRoot)(&config)

	return NewConfigClient(config, options...)
}

// ConfigClient ...
type ConfigClient struct {
	Address         string
	Concurrency     float64
	DeployDataDir   string        `yaml:"deployDataDir"`
	DeployTimeout   time.Duration `yaml:"deployTimeout"`
	CredentialsMode string        `yaml:"credentialsSource"`
	CredentialsDir  string        `yaml:"credentialsDir"`
	CA              string
	ServerName      string
	Environment     string
}

// Connect to the address in the config client.
func (t ConfigClient) Connect(creds credentials.TransportCredentials, options ...ConnectOption) (client Client, d Dialer, c clustering.Cluster, err error) {
	var (
		details ConnectResponse
	)

	conn := newConnect(options...)

	if client, d, details, err = t.connect(creds); err != nil {
		return client, d, c, err
	}

	if c, err = clusterConnect(details, conn.clustering.Options, conn.clustering.Bootstrap); err != nil {
		return client, d, c, err
	}

	return client, d, c, nil
}

func (t ConfigClient) connect(creds credentials.TransportCredentials) (c Client, d Dialer, details ConnectResponse, err error) {
	opts := DefaultDialerOptions(grpc.WithTransportCredentials(creds))
	d = NewDialer(opts...)
	if c, err = AddressProxyDialQuorum(t.Address, opts...); err != nil {
		return c, d, details, err
	}

	if details, err = c.Connect(); err != nil {
		return c, d, details, err
	}

	return c, d, details, err
}

// LoadConfig create a new configuration from the specified path using the current
// configuration as the default values for the new configuration.
func (t ConfigClient) LoadConfig(path string) (ConfigClient, error) {
	if err := bw.ExpandAndDecodeFile(path, &t); err != nil {
		return t, err
	}

	return t, nil
}

// Partitioner ...
func (t ConfigClient) Partitioner() (_ bw.Partitioner) {
	return bw.PartitionFromFloat64(t.Concurrency)
}

func clusterConnect(details ConnectResponse, copts []clustering.Option, bopts []clustering.BootstrapOption) (c clustering.Cluster, err error) {
	keyring, err := memberlist.NewKeyring([][]byte{}, details.Secret)
	if err != nil {
		return c, errors.Wrap(err, "failed to create keyring")
	}

	copts = append([]clustering.Option{
		clustering.OptionBindPort(0),
		clustering.OptionLogOutput(ioutil.Discard),
		clustering.OptionKeyring(keyring),
	}, copts...)

	if c, err = clustering.NewOptions(copts...).NewCluster(); err != nil {
		return c, errors.Wrap(err, "failed to join cluster")
	}

	bopts = append([]clustering.BootstrapOption{
		clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(1)),
		clustering.BootstrapOptionAllowRetry(clustering.UnlimitedAttempts),
		clustering.BootstrapOptionPeeringStrategies(
			BootstrapPeers(details.Quorum...),
		),
	}, bopts...)

	if err = clustering.Bootstrap(context.Background(), c, bopts...); err != nil {
		return c, errors.Wrap(err, "failed to connect to cluster")
	}

	return c, nil
}

func (t ConfigClient) creds() (credentials.TransportCredentials, error) {
	var (
		err  error
		conf *tls.Config
	)

	if conf, err = t.BuildClient(); err != nil {
		return nil, err
	}

	return credentials.NewTLS(conf), nil
}

// BootstrapPeers converts a list of Peers into a list of addresses to bootstrap from.
func BootstrapPeers(peers ...*Peer) peering.Static {
	speers := make([]string, 0, len(peers))
	for _, p := range peers {
		speers = append(speers, SWIMAddress(*p))
	}

	return peering.NewStatic(speers...)
}

// NewConfig creates a default configuration.
func NewConfig(options ...ConfigOption) Config {
	c := Config{
		Name:              systemx.HostnameOrLocalhost(),
		Root:              filepath.Join("/", "var", "cache", bw.DefaultDir),
		KeepN:             3,
		SnapshotFrequency: time.Hour,
		MinimumNodes:      3,
		BootstrapAttempts: math.MaxInt32,
		DNSBind: dnsBind{
			TTL:       60,
			Frequency: time.Hour,
		},
	}

	newTLSAgent(certificatecache.DefaultTLSCredentialsRoot, "")(&c)

	for _, opt := range options {
		opt(&c)
	}

	return c
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
			Port: bw.DefaultRPCPort,
		}),
		ConfigOptionSWIM(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultSWIMPort,
		}),
		ConfigOptionRaft(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultRaftPort,
		}),
		ConfigOptionTorrent(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultTorrentPort,
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

// ConfigOptionTorrent sets the Torrent address to bind.
func ConfigOptionTorrent(p *net.TCPAddr) ConfigOption {
	return func(c *Config) {
		c.TorrentBind = p
	}
}

// Config - configuration for agent processes.
type Config struct {
	Name              string
	Root              string        // root directory to store long term data.
	KeepN             int           `yaml:"keepN"`
	MinimumNodes      int           `yaml:"minimumNodes"`
	BootstrapAttempts int           `yaml:"bootstrapAttempts"`
	SnapshotFrequency time.Duration `yaml:"snapshotFrequency"`
	RPCBind           *net.TCPAddr
	RaftBind          *net.TCPAddr
	SWIMBind          *net.TCPAddr
	ClusterTokens     []string `yaml:"clusterTokens"`
	ServerName        string
	CA                string
	CredentialsMode   string `yaml:"credentialsSource"`
	CredentialsDir    string `yaml:"credentialsDir"`
	TorrentBind       *net.TCPAddr
	DNSBind           dnsBind  `yaml:"dnsBind"`
	DNSBootstrap      []string `yaml:"dnsBootstrap"`
	AWSBootstrap      struct {
		AutoscalingGroups []string `yaml:"autoscalingGroups"` // additional autoscaling groups to check for instances.
	} `yaml:"awsBootstrap"`
}

type dnsBind struct {
	TTL       uint32 // TTL for the generated records.
	Frequency time.Duration
}

// Peer - builds the Peer information from the configuration.
func (t Config) Peer() Peer {
	return Peer{
		Status:      Peer_Node,
		Name:        t.Name,
		Ip:          t.RPCBind.IP.String(),
		RPCPort:     uint32(t.RPCBind.Port),
		RaftPort:    uint32(t.RaftBind.Port),
		SWIMPort:    uint32(t.SWIMBind.Port),
		TorrentPort: uint32(t.TorrentBind.Port),
	}
}

// Keyring - returns the hash of the Secret.
func (t Config) Keyring() (ring *memberlist.Keyring, err error) {
	var (
		tokens [][]byte
	)

	for _, token := range t.ClusterTokens {
		hashed := sha256.Sum256([]byte(token))
		tokens = append(tokens, hashed[:])
	}

	switch len(tokens) {
	case 0:
		hashed := sha256.Sum256([]byte(t.ServerName))
		return memberlist.NewKeyring([][]byte{}, hashed[:])
	case 1:
		return memberlist.NewKeyring([][]byte{}, tokens[0])
	default:
		return memberlist.NewKeyring(tokens[1:], tokens[0])
	}
}
