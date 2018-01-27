// Package commandutils provides common utility functions for CLI interfaces.
package commandutils

import (
	"context"
	"log"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/x/systemx"
	"github.com/pkg/errors"
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

// LoadConfiguration loads the configuration for the given environment.
func LoadConfiguration(environment string) (agent.ConfigClient, error) {
	path := filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), environment)
	log.Println("loading configuration", path)
	return agent.DefaultConfigClient(agent.CCOptionTLSConfig(environment)).LoadConfig(path)
}

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

	defaultPeers = append(
		defaultPeers,
		peering.DNS{
			Port:  conf.SWIMBind.Port,
			Hosts: conf.DNSBootstrap,
		},
		peering.AWSAutoscaling{
			Port:               conf.SWIMBind.Port,
			SupplimentalGroups: conf.AWSBootstrap.AutoscalingGroups,
		},
	)

	joins := clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(conf.MinimumNodes))
	attempts := clustering.BootstrapOptionAllowRetry(clustering.MaximumAttempts(conf.BootstrapAttempts))
	peerings := clustering.BootstrapOptionPeeringStrategies(defaultPeers...)

	if err = clustering.Bootstrap(ctx, c, peerings, joins, attempts); err != nil {
		return c, errors.Wrap(err, "failed to bootstrap cluster")
	}

	return c, nil
}
