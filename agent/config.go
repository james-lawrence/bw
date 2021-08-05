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

// CCOptionInsecure insecure tls configuration
func CCOptionInsecure(b bool) ConfigClientOption {
	return func(c *ConfigClient) {
		c.Credentials.Insecure = b
	}
}

// CCOptionAddress set address for the configuration.
func CCOptionAddress(s string) ConfigClientOption {
	return func(c *ConfigClient) {
		c.Address = net.JoinHostPort(s, strconv.Itoa(bw.DefaultP2PPort))
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

// ExampleConfigClient creates an example configuration.
func ExampleConfigClient(options ...ConfigClientOption) ConfigClient {
	config := ConfigClient{
		Address:       systemx.HostnameOrLocalhost(),
		DeployDataDir: bw.LocateDeployspace(bw.DefaultDeployspaceDir),
		DeployPrompt:  "are you sure you want to deploy? (remove this field to disable the prompt)",
		DeployTimeout: bw.DefaultDeployTimeout,
	}

	ConfigClientTLS(bw.DefaultEnvironmentName)(&config)

	return NewConfigClient(config, options...)
}

// ConfigClient ...
type ConfigClient struct {
	root            string `yaml:"-"` // filepath of the configuration on disk.
	Address         string // cluster address
	Concurrency     float64
	DeployDataDir   string        `yaml:"deployDataDir"`
	DeployTimeout   time.Duration `yaml:"deployTimeout"`
	DeployPrompt    string        `yaml:"deployPrompt"`      // used to prompt before a deploy is started, useful for deploying to sensitive systems like production.
	CredentialsMode string        `yaml:"credentialsSource"` // deprecated
	CredentialsDir  string        `yaml:"credentialsDir"`    // deprecated
	Credentials     struct {
		Mode      string `yaml:"source"`
		Directory string `yaml:"directory"`
		Insecure  bool   `yaml:"-"`
	} `yaml:"credentials"`
	CA          string
	ServerName  string
	Environment string
}

// LoadConfig create a new configuration from the specified path using the current
// configuration as the default values for the new configuration.
func (t ConfigClient) LoadConfig(path string) (ConfigClient, error) {
	if err := bw.ExpandAndDecodeFile(path, &t); err != nil {
		return t, err
	}

	t.root = filepath.Dir(path)

	return t, nil
}

// Dir path to the configuration on disk
func (t ConfigClient) Dir() string {
	return t.root
}

// Partitioner ...
func (t ConfigClient) Partitioner() (_ bw.Partitioner) {
	return bw.PartitionFromFloat64(t.Concurrency)
}

// BootstrapPeers converts a list of Peers into a list of addresses to bootstrap from.
func BootstrapPeers(peers ...*Peer) peering.Static {
	speers := make([]string, 0, len(peers))
	for _, p := range peers {
		speers = append(speers, SWIMAddress(p))
	}

	return peering.NewStatic(speers...)
}

// NewConfig creates a default configuration.
func NewConfig(options ...ConfigOption) Config {
	c := Config{
		Name:              systemx.HostnameOrLocalhost(),
		Root:              bw.DefaultCacheDirectory(),
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

	newTLSAgent(bw.DefaultEnvironmentName)(&c)

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
		ConfigOptionP2P(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultP2PPort,
		}),
		ConfigOptionAutocert(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultP2PPort,
		}),
		ConfigOptionRPC(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultP2PPort,
		}),
		ConfigOptionSWIM(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultP2PPort,
		}),
		ConfigOptionRaft(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultP2PPort,
		}),
		ConfigOptionTorrent(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultP2PPort,
		}),
		ConfigOptionDiscovery(&net.TCPAddr{
			IP:   ip,
			Port: bw.DefaultP2PPort,
		}),
	)
}

// ConfigOptionP2P sets the libp2p address to bind.
func ConfigOptionP2P(p *net.TCPAddr) ConfigOption {
	return func(c *Config) {
		c.P2PBind = p
	}
}

// ConfigOptionAutocert sets the autocert address to bind.
func ConfigOptionAutocert(p *net.TCPAddr) ConfigOption {
	return func(c *Config) {
		c.AutocertBind = p
	}
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

// ConfigOptionTorrent sets the address for the torrent service address to bind.
func ConfigOptionTorrent(p *net.TCPAddr) ConfigOption {
	return func(c *Config) {
		c.TorrentBind = p
	}
}

// ConfigOptionDiscovery sets the address for the discovery service to bind.
func ConfigOptionDiscovery(p *net.TCPAddr) ConfigOption {
	return func(c *Config) {
		c.DiscoveryBind = p
	}
}

// ConfigOptionName set the name of the agent.
func ConfigOptionName(name string) ConfigOption {
	return func(c *Config) {
		c.Name = name
	}
}

type bootstrap struct {
	Attempts         int    `yaml:"attempts"`
	ReadOnly         bool   `yaml:"readonly"`
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
	AutocertBind      *net.TCPAddr
	TorrentBind       *net.TCPAddr
	P2PBind           *net.TCPAddr
	ClusterTokens     []string `yaml:"clusterTokens"`
	ServerName        string
	CA                string `yaml:"ca"`
	CredentialsMode   string `yaml:"credentialsSource"` // deprecated
	CredentialsDir    string `yaml:"credentialsDir"`    // deprecated
	Credentials       struct {
		Mode      string `yaml:"source"`
		Directory string `yaml:"directory"`
	} `yaml:"credentials"`
	DNSBind      dnsBind  `yaml:"dnsBind"`
	DNSBootstrap []string `yaml:"dnsBootstrap"`
	AWSBootstrap struct {
		AutoscalingGroups []string `yaml:"autoscalingGroups"` // additional autoscaling groups to check for instances.
	} `yaml:"awsBootstrap"`
}

func (t Config) Sanitize() Config {
	dup := t
	dup.ClusterTokens = []string{}
	return dup
}

// EnsureDefaults values after configuration load
func (t Config) EnsureDefaults() Config {
	if t.CredentialsDir == "" {
		t.CredentialsDir = filepath.Join(t.Root, bw.DefaultDirAgentCredentials)
	}

	if t.Credentials.Directory == "" {
		t.Credentials.Directory = filepath.Join(t.Root, bw.DefaultDirAgentCredentials)
	}

	if t.CA == "" {
		t.CA = filepath.Join(t.CredentialsDir, bw.DefaultTLSCertCA)
	}

	return t
}

type dnsBind struct {
	TTL       uint32 // TTL for the generated records.
	Frequency time.Duration
}

// Clone the config applying any provided options.
func (t Config) Clone(options ...ConfigOption) Config {
	for _, opt := range options {
		opt(&t)
	}

	return t
}

// Peer - builds the Peer information from the configuration.
func (t Config) Peer() *Peer {
	return &Peer{
		Status:        Peer_Node,
		Name:          t.Name,
		Ip:            t.RPCBind.IP.String(),
		AutocertPort:  uint32(t.AutocertBind.Port),
		RPCPort:       uint32(t.RPCBind.Port),
		RaftPort:      uint32(t.RaftBind.Port),
		SWIMPort:      uint32(t.SWIMBind.Port),
		TorrentPort:   uint32(t.TorrentBind.Port),
		DiscoveryPort: uint32(t.DiscoveryBind.Port),
		P2PPort:       uint32(t.P2PBind.Port),
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
