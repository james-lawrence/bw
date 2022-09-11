package clustering

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
)

// ErrPeeringOptionsExhausted returned by bootstrap methods when the strategies for peering have been exhausted.
var ErrPeeringOptionsExhausted = fmt.Errorf("ran out of peering options, unable to locate peers")

// BootstrapOption option for bootstrapping a clusters
type BootstrapOption func(*bootstrap)

// strategy for performing joins. provides the number of peers
// within the cluster that was joined, or an error.
// should return true if the join was considered successful.
type joinStrategy func(peers int) bool

type backoff interface {
	Backoff(attempt int) time.Duration
}

type backoffDefault struct{}

func (t backoffDefault) Backoff(int) time.Duration {
	return 5 * time.Second
}

// AllowSingleNode ...
func AllowSingleNode(peers int) bool {
	return true
}

// MinimumPeers ...
func MinimumPeers(minimum int) func(int) bool {
	return func(peers int) bool {
		return peers >= minimum
	}
}

type allowRetry func(attempts int) bool

// MaximumAttempts ...
func MaximumAttempts(max int) func(int) bool {
	return func(attempt int) bool {
		return attempt < max
	}
}

// UnlimitedAttempts ...
func UnlimitedAttempts(attempt int) bool {
	return true
}

// BootstrapOptionJoinStrategy - strategy to use to determine if a join
// was successful.
func BootstrapOptionJoinStrategy(s joinStrategy) BootstrapOption {
	return func(b *bootstrap) {
		b.JoinStrategy = s
	}
}

// BootstrapOptionAllowRetry - strategy to use to determine if another attempt
// should be made at joining the cluster.
func BootstrapOptionAllowRetry(s allowRetry) BootstrapOption {
	return func(b *bootstrap) {
		b.AllowRetry = s
	}
}

// BootstrapOptionPeeringStrategies - set the strategies for peering.
func BootstrapOptionPeeringStrategies(p ...Source) BootstrapOption {
	return func(b *bootstrap) {
		b.Peering = p
	}
}

// BootstrapOptionBackoff - backoff strategy to use.
func BootstrapOptionBackoff(s backoff) BootstrapOption {
	return func(b *bootstrap) {
		b.Backoff = s
	}
}

func BootstrapOptionBanned(b ...string) BootstrapOption {
	banned := make(map[string]struct{}, len(b))
	for _, n := range b {
		banned[n] = struct{}{}
	}

	return func(b *bootstrap) {
		b.Banned = banned
	}
}

type bootstrap struct {
	Backoff      backoff
	AllowRetry   allowRetry
	JoinStrategy joinStrategy
	Peering      []Source
	Banned       map[string]struct{}
}

func (t bootstrap) retrieve(ctx context.Context, s Source) (peers []string, err error) {
	pctx, done := context.WithTimeout(ctx, time.Minute)
	defer done()
	return s.Peers(pctx)
}

func (t bootstrap) collect(ctx context.Context, sources ...Source) (peers []string, err error) {
	for _, s := range t.Peering {
		log.Printf("%T: locating peers\n", s)
		localpeers, localerr := t.retrieve(ctx, s)
		if localerr != nil {
			err = errorsx.Compact(err, localerr)
			log.Printf("failed to load peers: %T: %s\n", s, localerr)
			continue
		}

		log.Printf("%T: located %d peers\n", s, len(localpeers))
		peers = append(peers, localpeers...)
	}

	dedup := make(map[string]bool, len(peers))
	for _, p := range peers {
		if _, ok := dedup[p]; ok {
			continue
		}

		if _, ok := t.Banned[p]; ok {
			log.Println("skipping banned peer", p)
			continue
		}

		dedup[p] = true
	}

	peers = peers[:0]
	for k := range dedup {
		peers = append(peers, k)
	}

	return peers, err
}

func newBootstrap(options ...BootstrapOption) bootstrap {
	b := bootstrap{
		Backoff:      backoffDefault{},
		AllowRetry:   MaximumAttempts(100),
		JoinStrategy: MinimumPeers(1),
		Banned:       make(map[string]struct{}),
	}

	for _, opt := range options {
		opt(&b)
	}

	return b
}

// Bootstrap - bootstraps the provided cluster using the options provided.
func Bootstrap(ctx context.Context, c Joiner, options ...BootstrapOption) (err error) {
	var (
		joined int
		peers  []string
	)

	max := func(a, b int) int {
		if a < b {
			return b
		}
		return a
	}

	b := newBootstrap(options...)

	for attempts := 0; ; attempts++ {
		peers, _ = b.collect(ctx, b.Peering...)

		log.Printf("located %d peers: %s\n", len(peers), spew.Sdump(peers))

		if joined, err = c.Join(peers...); err != nil {
			log.Println(errors.Wrap(err, "failed to join peers"))
			continue
		}

		if len(peers) > 6 {
			rand.Shuffle(len(peers), func(i int, j int) {
				peers[i], peers[j] = peers[j], peers[i]
			})
			peers = peers[:6]
			log.Printf("reduced to %d peers: %s\n", len(peers), spew.Sdump(peers))
		}

		// if members > 1, then another node discovered us while we were
		// attempting to join the cluster.
		joined = max(joined, len(c.Members()))

		log.Println("joined", joined, "peers")

		if b.JoinStrategy(joined) {
			return nil
		}

		if !b.AllowRetry(attempts) {
			break
		}

		time.Sleep(b.Backoff.Backoff(attempts))
	}

	if joined == 0 {
		return ErrPeeringOptionsExhausted
	}

	return nil
}

// Peers converts the peers into an array of host:port.
func Peers(c Rendezvous) []string {
	const key = "d989d44e-c327-41ef-9810-14a3768f2dc7"
	peers := c.GetN(10, []byte(key))
	list := make([]string, 0, len(peers))
	for _, peer := range peers {
		list = append(list, net.JoinHostPort(peer.Addr.String(), strconv.Itoa(int(peer.Port))))
	}
	return list
}
