// Package proxy handles being able to run a deploy from one of the members of the cluster.
// allowing for a deploy to be initiated by a local client and then continued even if that client disconnects.
package proxy

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"google.golang.org/grpc"
)

type dispatcher interface {
	Dispatch(...agent.Message) error
}

func check(options ...grpc.DialOption) func(n agent.Peer) error {
	return func(n agent.Peer) (err error) {
		var (
			c    agent.Client
			info agent.Status
		)

		if c, err = agent.Dial(agent.RPCAddress(n), options...); err != nil {
			return err
		}

		defer c.Close()

		if info, err = c.Info(); err != nil {
			return err
		}

		return deployment.NewStatus(info.Peer.Status)
	}
}

func deploy(info agent.Archive, options ...grpc.DialOption) func(n agent.Peer) error {
	return func(n agent.Peer) (err error) {
		var (
			c agent.Client
		)

		if c, err = agent.Dial(agent.RPCAddress(n), options...); err != nil {
			return err
		}
		defer c.Close()

		if err = c.Deploy(info); err != nil {
			return err
		}

		return nil
	}
}
