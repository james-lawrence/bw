// Package agentutil provides common utilities for an agent
// Examples include removing old files from directories etc.
package agentutil

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"golang.org/x/time/rate"

	"github.com/pkg/errors"
)

const (
	// ErrNoDeployments ...
	ErrNoDeployments = errorsx.String("no deployments found")
	// ErrFailedDeploymentQuorum ...
	ErrFailedDeploymentQuorum = errorsx.String("unable to achieve latest deployment quorum")
	// ErrActiveDeployment ...
	ErrActiveDeployment = errorsx.String("deployment in progress")
	// ErrDifferentDeployment when two deployments have different IDs.
	ErrDifferentDeployment = errorsx.String("deployments are different")
)

type dialer interface {
	Dial(agent.Peer) (zeroc agent.Client, err error)
}

type cluster interface {
	Local() agent.Peer
	Peers() []agent.Peer
	Quorum() []agent.Peer
}

// Cleaner interface for cleaning workspace directories.
type Cleaner interface {
	Clean(...FileInfo) error
}

// FileInfo contains os.FileInfo and the absolute path of a file
type FileInfo struct {
	Path string
	Info os.FileInfo
}

// MaybeClean uses a cleaner to clean a set of FileInfo if no error
// is provided.
func MaybeClean(c Cleaner) func([]FileInfo, error) error {
	return func(infos []FileInfo, err error) error {
		if err != nil {
			log.Println("MaybeClean err", err)
			return err
		}

		return c.Clean(infos...)
	}
}

// Dirs retrieve directories under the root.
func Dirs(root string) ([]FileInfo, error) {
	dirs := make([]FileInfo, 0, 10)
	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip the root directory itself.
		if root != path && info.IsDir() {
			dirs = append(dirs, FileInfo{Path: path, Info: info})
			return filepath.SkipDir
		}

		return nil
	}

	if err := filepath.Walk(root, walker); err != nil {
		return dirs, errors.WithStack(err)
	}

	return dirs, nil
}

type cleanerFunc func(...FileInfo) error

func (t cleanerFunc) Clean(infos ...FileInfo) error {
	return t(infos...)
}

func _cleanerFunc(n int, sorter func(i, j FileInfo) bool) Cleaner {
	return cleanerFunc(func(infos ...FileInfo) (err error) {
		if len(infos) <= n {
			return nil
		}

		sort.Slice(infos, func(i, j int) bool {
			return sorter(infos[i], infos[j])
		})

		for _, info := range infos[n:] {
			if err = os.RemoveAll(info.Path); err != nil {
				return errors.WithStack(err)
			}
		}

		return nil
	})
}

// KeepOldestN keeps oldest n directories.
func KeepOldestN(n int) Cleaner {
	return _cleanerFunc(n, func(i, j FileInfo) bool {
		return i.Info.ModTime().Before(j.Info.ModTime())
	})
}

// KeepNewestN keeps newest n directories.
func KeepNewestN(n int) Cleaner {
	return _cleanerFunc(n, func(i, j FileInfo) bool {
		return !i.Info.ModTime().Before(j.Info.ModTime())
	})
}

// WatchEvents connects to the event stream of the cluster using the provided
// peer as a proxy.
func WatchEvents(local, proxy agent.Peer, d dialer, events chan agent.Message) {
	rl := rate.NewLimiter(rate.Every(time.Second), 1)
	for {
		var (
			err error
			qc  agent.Client
		)

		if err = rl.Wait(context.Background()); err != nil {
			events <- LogError(local, errors.Wrap(err, "failed to wait during rate limiting"))
			continue
		}

		if qc, err = agent.NewProxyDialer(d).Dial(proxy); err != nil {
			events <- LogError(local, errors.Wrap(err, "events dialer failed to connect"))
			continue
		}

		if err = qc.Watch(context.Background(), events); err != nil {
			events <- LogError(local, errors.Wrap(err, "connection lost, reconnecting"))
			continue
		}
	}
}

// WatchClusterEvents pushes events into the provided channel for the given cluster.
func WatchClusterEvents(ctx context.Context, d agent.Dialer, c cluster, events chan agent.Message) {
	rl := rate.NewLimiter(rate.Every(time.Second), 3)
	for {
		var (
			err   error
			qc    agent.Client
			local = c.Local()
		)

		if qc != nil {
			logx.MaybeLog(qc.Close())
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		if err = rl.Wait(ctx); err != nil {
			events <- LogError(local, errors.Wrap(err, "failed to wait during rate limiting"))
			continue
		}

		if qc, err = agent.NewQuorumDialer(d).Dial(c); err != nil {
			events <- LogError(local, errors.Wrap(err, "events dialer failed to connect"))
			continue
		}

		if err = qc.Watch(ctx, events); err != nil {
			events <- LogError(local, errors.Wrap(err, "connection lost, reconnecting"))
			continue
		}
	}
}

// SameArchive checks if two archives have the same deployment ID.
// if both are nil, they are considered the same deployment.
// if one is nil, they are considered different.
// otherwise DeploymentID must match to be considered the same deployment.
func SameArchive(a, b *agent.Archive) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return bytes.Compare(a.DeploymentID, b.DeploymentID) == 0
}

// IsActiveDeployment checks if the error is an active deployment.
func IsActiveDeployment(err error) bool {
	return errors.Cause(err) == ErrActiveDeployment
}

// IsNoDeployments checks if the error is no deployments.
func IsNoDeployments(err error) bool {
	return errors.Cause(err) == ErrNoDeployments
}
