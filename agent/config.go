package agent

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie"
	cp "bitbucket.org/jatone/bearded-wookie/cluster"
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

// Connect to the cluster.
func (t ConfigClient) Connect(copts []clustering.Option, bopts []clustering.BootstrapOption) (creds credentials.TransportCredentials, client Client, c clustering.Cluster, err error) {
	var (
		secret []byte
		peers  []string
		conf   *tls.Config
	)

	if conf, err = t.TLSConfig.BuildClient(); err != nil {
		return creds, client, c, err
	}

	creds = credentials.NewTLS(conf)

	if client, err = DialClient(t.Address, grpc.WithTransportCredentials(creds)); err != nil {
		return creds, client, c, err
	}

	if peers, secret, err = client.Credentials(); err != nil {
		return creds, client, c, err
	}

	copts = append([]clustering.Option{
		clustering.OptionBindPort(0),
		clustering.OptionDelegate(cp.NewLocal(cp.BitFieldMerge([]byte(nil), cp.Deploy))),
		clustering.OptionAliveDelegate(cp.AliveDefault{}),
		clustering.OptionLogOutput(os.Stderr),
		clustering.OptionSecret(secret),
	}, copts...)

	if c, err = clustering.NewOptions(copts...).NewCluster(); err != nil {
		return creds, client, c, errors.Wrap(err, "failed to join cluster")
	}

	log.Println("peers located", peers)
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
		return creds, client, c, errors.Wrap(err, "failed to connect to cluster")
	}

	return creds, client, c, nil
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
