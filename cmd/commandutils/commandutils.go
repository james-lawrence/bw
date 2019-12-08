// Package commandutils provides common utility functions for CLI interfaces.
package commandutils

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	cc "github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/internal/x/systemx"
)

// NewClientPeer create a client peer.
func NewClientPeer(options ...agent.PeerOption) (p agent.Peer) {
	return agent.NewPeerFromTemplate(
		agent.Peer{
			Name:   bw.MustGenerateID().String(),
			Ip:     systemx.HostnameOrLocalhost(),
			Status: agent.Peer_Client,
		},
		append(options, agent.PeerOptionStatus(agent.Peer_Client))...,
	)
}

// ReadConfiguration reads the configuration for the given environment.
func ReadConfiguration(environment string) (config agent.ConfigClient, err error) {
	path := filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), environment)
	log.Println("loading configuration", path)
	return agent.DefaultConfigClient(agent.CCOptionTLSConfig(environment)).LoadConfig(path)
}

// LoadConfiguration loads the configuration for the given environment.
func LoadConfiguration(environment string, options ...agent.ConfigClientOption) (config agent.ConfigClient, err error) {
	path := filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), environment)
	log.Println("loading configuration", path, bw.DefaultCacheDirectory())
	if config, err = agent.DefaultConfigClient(append(options, agent.CCOptionTLSConfig(environment))...).LoadConfig(path); err != nil {
		return config, errors.Wrap(err, "configuration load failed")
	}

	// load or create credentials.
	if err = cc.FromConfig(config.CredentialsDir, config.CredentialsMode, path, cc.NewRefreshClient()); err != nil {
		return config, err
	}

	return config, err
}

// NewClusterDialer dial a cluster based on the configuration.
func NewClusterDialer(conf agent.Config, options ...clustering.Option) clustering.Dialer {
	options = append(
		[]clustering.Option{
			clustering.OptionBindAddress(conf.SWIMBind.IP.String()),
			clustering.OptionBindPort(conf.SWIMBind.Port),
			clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		},
		options...,
	)

	return clustering.NewDialer(options...)
}

// ClusterJoin connects to a cluster.
func ClusterJoin(ctx context.Context, conf agent.Config, dialer clustering.Dialer, defaultPeers ...clustering.Source) (clustering.Cluster, error) {
	var (
		err error
		c   clustering.Cluster
	)

	log.Println("connecting to cluster")
	defer log.Println("connection to cluster complete")

	if c, err = dialer.Dial(); err != nil {
		return c, err
	}

	joins := clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(conf.MinimumNodes))
	attempts := clustering.BootstrapOptionAllowRetry(clustering.MaximumAttempts(conf.Bootstrap.Attempts))
	peerings := clustering.BootstrapOptionPeeringStrategies(defaultPeers...)

	if err = clustering.Bootstrap(ctx, c, peerings, joins, attempts); err != nil {
		return c, errors.Wrap(err, "failed to bootstrap cluster")
	}

	return c, nil
}

// DebugLog return a logger that is either enabled or disabled for debugging purposes.
func DebugLog(debug bool) *log.Logger {
	if debug {
		return log.New(os.Stderr, log.Prefix(), log.Flags())
	}

	return log.New(ioutil.Discard, log.Prefix(), log.Flags())
}
