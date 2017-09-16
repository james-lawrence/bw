package agent

import (
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
	)

	if creds, err = t.TLSConfig.BuildClient(); err != nil {
		return creds, client, c, err
	}

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
		clustering.OptionLogger(os.Stderr),
		clustering.OptionSecret(secret),
	}, copts...)

	if c, err = clustering.NewOptions(copts...).NewCluster(); err != nil {
		return creds, client, c, errors.Wrap(err, "failed to join cluster")
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
		return creds, client, c, errors.Wrap(err, "failed to connect to cluster")
	}

	return creds, client, c, nil
}

// NewConfig creates a default configuration.
func NewConfig() Config {
	return Config{
		Root:      filepath.Join("/", "var", "cache", bw.DefaultDir),
		KeepN:     3,
		TLSConfig: NewTLSAgent(DefaultTLSCredentialsRoot, ""),
		Storage:   uploads.Config{Backend: "local"},
		Cluster: clusteringConfig{
			SnapshotFrequency: time.Hour,
		},
	}
}

type clusteringConfig struct {
	SnapshotFrequency time.Duration
}

// Config - configuration for the agent.
type Config struct {
	Name      string
	Root      string // root directory to store long term data.
	KeepN     int    `yaml:"keepN"`
	Storage   uploads.Config
	TLSConfig TLSConfig
	Cluster   clusteringConfig
}
