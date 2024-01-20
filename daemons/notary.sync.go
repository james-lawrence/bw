package daemons

import (
	"context"
	"log"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"
)

func SyncAuthorizations(dctx Context) {
	var (
		err error
		b   *bloom.BloomFilter
	)

	if b, err = dctx.NotaryStorage.Bloomfilter(dctx.Context); err != nil {
		log.Println("unable to generate bloomfilter for authorization synchronization", err)
		return
	}

	syncpeer := func(p *agent.Peer) error {
		log.Println("syncing credentials initiated", agent.RPCAddress(p))
		defer log.Println("syncing credentials completed", agent.RPCAddress(p))

		// Notary Subscriptions to node events. syncs authorization between agents
		req, err := notary.NewSyncRequest(b)
		if err != nil {
			return errors.Wrap(err, "unable to generate request")
		}

		d := dialers.NewDirect(agent.RPCAddress(p), dctx.Dialer.Defaults()...)
		ctx, done := context.WithTimeout(dctx.Context, 5*time.Minute)
		conn, err := d.DialContext(ctx)
		done()

		if err != nil {
			log.Println("unable to connect", err)
			return errors.Wrapf(err, "unable to connect to peer %s", p.Ip)
		}

		client := notary.NewSyncClient(conn)
		stream, err := client.Stream(dctx.Context, req)
		if err != nil {
			return errors.Wrap(err, "stream creation failed")
		}

		log.Println("syncing credentials initiated", agent.RPCAddress(p))
		if err = notary.Sync(stream, b, dctx.NotaryStorage); err != nil {
			return errors.Wrap(err, "syncing credentials failed")
		}
		log.Println("syncing credentials completed", agent.RPCAddress(p))

		return nil
	}

	for _, p := range agent.RendezvousPeers(dctx.P2PPublicKey, dctx.Cluster) {
		if err := syncpeer(p); err != nil {
			log.Println("authorization sync failed", agent.RPCAddress(p), err)
		}
	}
}
