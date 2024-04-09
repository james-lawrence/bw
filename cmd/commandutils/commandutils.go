// Package commandutils provides common utility functions for CLI interfaces.
package commandutils

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	cc "github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/netx"
	"github.com/james-lawrence/bw/internal/systemx"
	"github.com/james-lawrence/bw/internal/tlsx"
)

// NewClientPeer create a client peer.
func NewClientPeer(options ...agent.PeerOption) (p *agent.Peer) {
	return agent.NewPeerFromTemplate(
		&agent.Peer{
			Name:   bw.MustGenerateID().String(),
			Ip:     systemx.HostnameOrLocalhost(),
			Status: agent.Peer_Client,
		},
		append(options, agent.PeerOptionStatus(agent.Peer_Client))...,
	)
}

// PersistAgentName to disk to prevent name changes from impacting the cluster.
func PersistAgentName(proto agent.Config) (c agent.Config, err error) {
	var (
		raw  []byte
		path = filepath.Join(proto.Root, "agent.name")
	)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return proto, os.WriteFile(path, []byte(proto.Name), 0600)
	}

	if raw, err = os.ReadFile(path); err != nil {
		return proto, errors.Wrap(err, "failed to read persisted agent name")
	}

	return proto.Clone(agent.ConfigOptionName(string(raw))), nil
}

// LoadAgentConfig - load the agent configuration from the provided file.
func LoadAgentConfig(path string, proto agent.Config) (c agent.Config, err error) {
	if err = bw.ExpandAndDecodeFile(path, &proto); err != nil {
		return c, err
	}
	return proto.EnsureDefaults(), nil
}

// LoadConfiguration loads the configuration for the given environment.
func LoadConfiguration(ctx context.Context, environment string, options ...agent.ConfigClientOption) (config agent.ConfigClient, err error) {
	var (
		d         dialers.Defaults
		tlsconfig *tls.Config
	)

	path := bw.LocateFirst(
		filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), environment, bw.DefaultClientConfig),
		filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), environment),
	)

	if _, err = os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return config, errorsx.UserFriendly(errors.Errorf("unknown environment: %s - %s", environment, path))
		}

		return config, err
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("loading configuration", path, bw.DefaultCacheDirectory())
	}

	if config, err = agent.DefaultConfigClient(append(options, agent.CCOptionTLSConfig(environment))...).LoadConfig(path); err != nil {
		return config, errors.Wrap(err, "configuration load failed")
	}

	if tlsconfig, err = cc.TLSGenClient(config); err != nil {
		return config, errors.Wrap(err, "failed to generate client TLS")
	}

	if d, err = dialers.DefaultDialer(config.Address, tlsx.NewDialer(tlsconfig)); err != nil {
		return config, errors.Wrap(err, "failed to create network dialer")
	}

	certpath := bw.LocateFirstInDir(
		config.Credentials.Directory,
		cc.DefaultTLSCertCA,
		cc.DefaultTLSCertServer,
		cc.DefaultTLSCertClient,
	)

	if err = discovery.CheckCredentials(ctx, config.Address, certpath, d); err != nil {
		if !grpcx.IsNotFound(err) {
			return config, errors.Wrap(err, "failed to check credentials")
		}

		errorsx.MaybeLog(os.Remove(filepath.Join(config.Credentials.Directory, cc.DefaultTLSCertCA)))
		errorsx.MaybeLog(os.Remove(bw.LocateFirstInDir(
			config.Credentials.Directory,
			cc.DefaultTLSCertServer,
			cc.DefaultTLSCertClient,
		)))
		errorsx.MaybeLog(os.Remove(bw.LocateFirstInDir(
			config.Credentials.Directory,
			cc.DefaultTLSKeyServer,
			cc.DefaultTLSKeyClient,
		)))
	}

	// load or create credentials.
	if err = cc.FromConfig(
		config.Credentials.Directory,
		config.Credentials.Mode,
		path,
		cc.NewRefreshClient(
			config.Credentials.Directory,
			config.Credentials.Insecure,
		),
	); err != nil {
		return config, errors.Wrapf(
			err,
			"unable to load credentials: %s %s %s",
			path,
			config.Credentials.Directory,
			config.Credentials.Mode,
		)
	}

	if envx.Boolean(false, bw.EnvLogsConfiguration) {
		log.Println("configuration", spew.Sdump(config))
	}

	return config, nil
}

// ReadConfiguration reads the configuration for the given environment.
func ReadConfiguration(environment string) (config agent.ConfigClient, err error) {
	// migrate environment to directory structure.
	path := filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), environment)
	if i, err := os.Stat(path); err == nil && !i.IsDir() {
		log.Println("detected old configuration migrating environment")
		dst := filepath.Join(path, bw.DefaultClientConfig)
		tmp := path + ".bak"
		err = errorsx.Compact(
			os.Rename(path, tmp),
			os.MkdirAll(path, 0700),
			os.Rename(tmp, dst),
		)

		if err != nil {
			return config, errorsx.UserFriendly(errors.Wrapf(err, "failed environment migration: %s - %s -> %s", environment, path, dst))
		}
	}

	path = filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), environment, bw.DefaultClientConfig)
	if _, err = os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return config, errorsx.UserFriendly(errors.Wrapf(err, "unknown environment: %s - %s", environment, path))
		}

		return config, err
	}

	log.Println("loading configuration", path)

	if config, err = agent.DefaultConfigClient(agent.CCOptionTLSConfig(environment)).LoadConfig(path); err != nil {
		return config, err
	}

	if envx.Boolean(false, bw.EnvLogsConfiguration) {
		log.Println("configuration", spew.Sdump(config))
	}

	return config, err
}

// ClusterJoin connects to a cluster.
func ClusterJoin(ctx context.Context, conf agent.Config, c clustering.Joiner, defaultPeers ...clustering.Source) (err error) {
	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("connecting to cluster")
		defer log.Println("connection to cluster complete")
	}

	joins := clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(conf.MinimumNodes))
	attempts := clustering.BootstrapOptionAllowRetry(clustering.MaximumAttempts(conf.Bootstrap.Attempts))
	peerings := clustering.BootstrapOptionPeeringStrategies(defaultPeers...)
	banned := clustering.BootstrapOptionBanned(
		append(
			netx.AddrToString(conf.AlternateBinds...),
			conf.P2PAdvertised.String(),
			conf.P2PBind.String(),
		)...,
	)
	if err = clustering.Bootstrap(ctx, c, peerings, joins, attempts, banned); err != nil {
		return errors.Wrap(err, "failed to bootstrap cluster")
	}

	return nil
}

// DebugLog return a logger that is either enabled or disabled for debugging purposes.
func DebugLog(debug bool) *log.Logger {
	if debug {
		return log.New(os.Stderr, log.Prefix(), log.Flags())
	}

	return log.New(io.Discard, log.Prefix(), log.Flags())
}

func RaftStoreFilepath(p string) (*raftboltdb.BoltStore, error) {
	sopts := raftboltdb.Options{
		Path: p,
	}
	return raftboltdb.New(sopts)
}
