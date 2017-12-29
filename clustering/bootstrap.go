package clustering

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
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

type bootstrap struct {
	Backoff      backoff
	AllowRetry   allowRetry
	JoinStrategy joinStrategy
	Peering      []Source
}

func newBootstrap(options ...BootstrapOption) bootstrap {
	b := bootstrap{
		Backoff:      backoffDefault{},
		AllowRetry:   MaximumAttempts(100),
		JoinStrategy: MinimumPeers(1),
	}

	for _, opt := range options {
		opt(&b)
	}

	return b
}

// Bootstrap - bootstraps the provided cluster using the options provided.
func Bootstrap(ctx context.Context, c cluster, options ...BootstrapOption) error {
	var (
		err      error
		joined   int
		peers    []string
		attempts int
	)

	max := func(a, b int) int {
		if a < b {
			return b
		}
		return a
	}

	b := newBootstrap(options...)

retry:

	for _, s := range b.Peering {
		// check if the join has been cancelled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Printf("%T: locating peers\n", s)
		if peers, err = s.Peers(); err != nil {
			log.Printf("failed to load peers: %T: %s\n", s, err)
			continue
		}

		log.Printf("%T: located %d peers\n", s, len(peers))
		if joined, err = c.Join(peers...); err != nil {
			log.Printf("failed to join peers: %T: %s\n", s, err)
			continue
		}

		if joined <= 1 {
			log.Printf("join succeeded but no new peers were located: %T\n", s)
			continue
		}

		log.Println("breaking out of loop")
		break
	}

	// if members > 1, then another node discovered us while we were
	// attempting to join the cluster.
	joined = max(joined, len(c.Members()))

	if b.JoinStrategy(joined) {
		return nil
	}

	if b.AllowRetry(attempts) {
		time.Sleep(b.Backoff.Backoff(attempts))
		attempts = attempts + 1
		goto retry
	}

	if joined == 0 {
		return ErrPeeringOptionsExhausted
	}

	return nil
}

// Peers converts the peers into an array of host:port.
func Peers(c cluster) []string {
	peers := c.Members()
	list := make([]string, 0, len(peers))
	for _, peer := range peers {
		list = append(list, net.JoinHostPort(peer.Addr.String(), strconv.Itoa(int(peer.Port))))
	}
	return list
}
