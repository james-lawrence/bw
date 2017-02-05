package serfdom

import (
	"io"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

type ClusterOption func(*memberlist.Config)

func COBindInterface(addr string) ClusterOption {
	return func(config *memberlist.Config) {
		config.BindAddr = addr
	}
}

func COBindPort(port int) ClusterOption {
	return func(config *memberlist.Config) {
		config.BindPort = port
	}
}

func COAdvertiseInterface(addr string) ClusterOption {
	return func(config *memberlist.Config) {
		config.AdvertiseAddr = addr
	}
}

func COAdvertisePort(port int) ClusterOption {
	return func(config *memberlist.Config) {
		config.AdvertisePort = port
	}
}

func CODelegate(delegate memberlist.Delegate) ClusterOption {
	return func(config *memberlist.Config) {
		config.Delegate = delegate
	}
}

func COAliveDelegate(delegate memberlist.AliveDelegate) ClusterOption {
	return func(config *memberlist.Config) {
		config.Alive = delegate
	}
}

func COEventsDelegate(delegate memberlist.EventDelegate) ClusterOption {
	return func(config *memberlist.Config) {
		config.Events = delegate
	}
}

func COConflictDelegate(delegate memberlist.ConflictDelegate) ClusterOption {
	return func(config *memberlist.Config) {
		config.Conflict = delegate
	}
}

func COLogger(out io.Writer) ClusterOption {
	return func(config *memberlist.Config) {
		config.LogOutput = out
	}
}

func New(name string, options ...ClusterOption) (*memberlist.Memberlist, error) {
	var (
		// config = memberlist.DefaultLocalConfig()
		config = memberlist.DefaultWANConfig()
	)

	config.Name = name
	config.Alive = aliveHandler{}

	for _, opt := range options {
		opt(config)
	}

	return memberlist.Create(config)
}

// GracefulShutdown gracefully leaves the cluster.
func GracefulShutdown(cluster *memberlist.Memberlist) error {
	var (
		err error
	)

	if err = errors.Wrap(cluster.Leave(5*time.Second), "failure to leave cluster"); err != nil {
		return err
	}

	if err = errors.Wrap(cluster.Shutdown(), "failure to shutdown node"); err != nil {
		return err
	}

	return nil
}

type ClusterFilter func(memberlist.Memberlist) []*memberlist.Node

// type Instance struct {
// 	serf.Member
// }
//
// func (t Instance) Name() string {
// 	return t.Member.Name
// }
//
// func (t Instance) IP() net.IP {
// 	return t.Member.Addr
// }
//
// func (t Instance) Tags() map[string]string {
// 	return t.Member.Tags
// }
