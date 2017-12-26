package agent

import (
	"crypto/sha256"
	"crypto/tls"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/storage"
	"github.com/james-lawrence/bw/x/systemx"
)

// ConfigClientOption options for the client configuration.
type ConfigClientOption func(*ConfigClient)

// CCOptionTLSConfig tls set to load for the client configuration.
func CCOptionTLSConfig(name string) ConfigClientOption {
	return func(c *ConfigClient) {
		c.TLSConfig = NewTLSClient(name)
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
		Address:   systemx.HostnameOrLocalhost(),
		TLSConfig: NewTLSClient(DefaultTLSCredentialsRoot),
	}

	return NewConfigClient(config, options...)
}

// ConfigClient ...
type ConfigClient struct {
	Address     string
	Concurrency float64
	TLSConfig
}

// Connect to the address in the config client.
func (t ConfigClient) Connect(options ...ConnectOption) (creds credentials.TransportCredentials, client Conn, c clustering.Cluster, err error) {
	var (
		details ConnectInfo
	)

	conn := newConnect(options...)

	if creds, client, details, err = t.connect(); err != nil {
		return creds, client, c, err
	}

	if c, err = clusterConnect(details, conn.clustering.Options, conn.clustering.Bootstrap); err != nil {
		return creds, client, c, err
	}

	return creds, client, c, nil
}

// ConnectLeader ...
func (t ConfigClient) ConnectLeader(options ...ConnectOption) (creds credentials.TransportCredentials, client Conn, c clustering.Cluster, err error) {
	var (
		details ConnectInfo
		success bool
		tmp     Conn
	)
	conn := newConnect(options...)

	if creds, tmp, details, err = t.connect(); err != nil {
		return creds, client, c, err
	}
	defer tmp.Close()

	for _, p := range details.Quorum {
		if client, err = Dial(RPCAddress(*p), grpc.WithTransportCredentials(creds)); err != nil {
			log.Println("failed to connect to peer", p.Name, p.Ip, err)
			continue
		}
		success = true
	}

	if !success {
		return creds, client, c, errors.New("failed to connect to a member of the quorum")
	}

	if c, err = clusterConnect(details, conn.clustering.Options, conn.clustering.Bootstrap); err != nil {
		return creds, client, c, err
	}

	return creds, client, c, nil
}

func (t ConfigClient) connect() (creds credentials.TransportCredentials, client Conn, details ConnectInfo, err error) {
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

func clusterConnect(details ConnectInfo, copts []clustering.Option, bopts []clustering.BootstrapOption) (c clustering.Cluster, err error) {
	copts = append([]clustering.Option{
		clustering.OptionBindPort(0),
		clustering.OptionLogOutput(os.Stderr),
		clustering.OptionSecret(details.Secret),
	}, copts...)

	if c, err = clustering.NewOptions(copts...).NewCluster(); err != nil {
		return c, errors.Wrap(err, "failed to join cluster")
	}

	bopts = append([]clustering.BootstrapOption{
		clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(1)),
		clustering.BootstrapOptionAllowRetry(clustering.UnlimitedAttempts),
		clustering.BootstrapOptionPeeringStrategies(
			peering.Closure(BootstrapPeers(details.Quorum...)),
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

// BootstrapPeers converts a list of Peers into a list of addresses to bootstrap from.
func BootstrapPeers(peers ...*Peer) func() ([]string, error) {
	speers := make([]string, 0, len(peers))
	for _, p := range peers {
		speers = append(speers, SWIMAddress(*p))
	}

	return func() ([]string, error) {
		return speers, nil
	}
}

// NewConfig creates a default configuration.
func NewConfig(options ...ConfigOption) Config {
	c := Config{
		Name:              systemx.HostnameOrLocalhost(),
		Root:              filepath.Join("/", "var", "cache", bw.DefaultDir),
		KeepN:             3,
		SnapshotFrequency: time.Hour,
		MinimumPeers:      3,
		BootstrapAttempts: math.MaxInt32,
		TLSConfig:         NewTLSAgent(DefaultTLSCredentialsRoot, ""),
	}

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

// Config - configuration for the
type Config struct {
	Name              string
	Root              string        // root directory to store long term data.
	KeepN             int           `yaml:"keepN"`
	MinimumPeers      int           `yaml:"minimumPeers"`
	BootstrapAttempts int           `yaml:"bootstrapAttempts"`
	SnapshotFrequency time.Duration `yaml:"snapshotFrequency"`
	RPCBind           *net.TCPAddr
	RaftBind          *net.TCPAddr
	SWIMBind          *net.TCPAddr
	Storage           storage.Config
	Secret            string
	TLSConfig         TLSConfig
}

// Peer - builds the Peer information from the configuration. by default
// a peer starts in the unknown state.
func (t Config) Peer() Peer {
	// TODO: have a separate advertise address for the IP field.
	return Peer{
		Status:   Peer_Ready,
		Name:     t.Name,
		Ip:       t.RPCBind.IP.String(),
		RPCPort:  uint32(t.RPCBind.Port),
		RaftPort: uint32(t.RaftBind.Port),
		SWIMPort: uint32(t.SWIMBind.Port),
	}
}

// Hash - returns the hash of the Secret.
func (t Config) Hash() (raw []byte, err error) {
	compute := sha256.New()

	if _, err = compute.Write([]byte(t.Secret)); err != nil {
		return raw, errors.WithStack(err)
	}

	return compute.Sum(nil), nil
}
