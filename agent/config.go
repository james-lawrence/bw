package agent

import (
	"crypto/sha256"
	"math"
	"net"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/memberlist"

	"github.com/james-lawrence/bw"
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
		c.Address = net.JoinHostPort(s, strconv.Itoa(bw.DefaultRPCPort))
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

	ConfigClientTLS(bw.DefaultEnvironmentName)(&config)

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
		Bootstrap: bootstrap{
			Attempts: math.MaxInt32,
		},
		DNSBind: dnsBind{
			TTL:       60,
			Frequency: time.Hour,
		},
	}

	newTLSAgent(bw.DefaultEnvironmentName, "")(&c)

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

type bootstrap struct {
	Attempts         int
	ArchiveDirectory string `yaml:"archiveDirectory"`
}

// Config - configuration for agent processes.
type Config struct {
	Name              string
	Root              string        // root directory to store long term data.
	KeepN             int           `yaml:"keepN"`
	MinimumNodes      int           `yaml:"minimumNodes"`
	Bootstrap         bootstrap     `yaml:"bootstrap"`
	SnapshotFrequency time.Duration `yaml:"snapshotFrequency"`
	DiscoveryBind     *net.TCPAddr
	RPCBind           *net.TCPAddr
	RaftBind          *net.TCPAddr
	SWIMBind          *net.TCPAddr
	ClusterTokens     []string `yaml:"clusterTokens"`
	ServerName        string
	CA                string `yaml:"ca"`
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
		Status:        Peer_Node,
		Name:          t.Name,
		Ip:            t.RPCBind.IP.String(),
		RPCPort:       uint32(t.RPCBind.Port),
		RaftPort:      uint32(t.RaftBind.Port),
		SWIMPort:      uint32(t.SWIMBind.Port),
		TorrentPort:   uint32(t.TorrentBind.Port),
		DiscoveryPort: uint32(t.DiscoveryBind.Port),
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
