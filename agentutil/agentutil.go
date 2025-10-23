// Package agentutil provides common utilities for an agent
// Examples include removing old files from directories etc.
package agentutil

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
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

type cluster interface {
	GetN(n int, key []byte) []*memberlist.Node
	Peers() []*agent.Peer
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
func WatchEvents(ctx context.Context, local *agent.Peer, d dialers.ContextDialer, events chan *agent.Message) {
	rl := rate.NewLimiter(rate.Every(10*time.Second), 1)
	var (
		err  error
		conn *grpc.ClientConn
	)

	for {
		if conn != nil {
			errorsx.Log(conn.Close())
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		if err = rl.Wait(ctx); err != nil {
			events <- agent.LogError(local, errors.Wrap(err, "failed to wait during rate limiting"))
			continue
		}

		if conn, err = d.DialContext(ctx); err != nil {
			events <- agent.LogError(local, errors.Wrap(err, "events dialer failed to connect"))
			continue
		}

		events <- agent.NewConnectionLog(local, agent.ConnectionEvent_Connected, fmt.Sprintf("connection established %s", conn.Target()))
		err = grpcx.Retry(ctx, func() error {
			if history, err := agent.NewQuorumClient(conn).History(ctx, &agent.HistoryRequest{}); err == nil {
				select {
				case events <- agent.NewLogHistoryFromMessages(local, history.Messages...):
				case <-ctx.Done():
					events <- agent.LogError(local, errors.Wrap(ctx.Err(), "unable to replay history"))
				}
				return nil
			} else {
				return err
			}
		}, codes.Unavailable)
		if err != nil {
			events <- agent.LogError(local, err)
		}

		if err = agent.NewDeployConn(conn).Watch(ctx, events); err != nil {
			msg := fmt.Sprintf("connection lost %s", conn.Target())
			err = errors.Wrap(err, msg)
			if envx.Boolean(false, bw.EnvLogsDeploy) {
				log.Println(err)
			}
			events <- agent.NewConnectionLog(local, agent.ConnectionEvent_Disconnected, msg)
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

	return bytes.Equal(a.DeploymentID, b.DeploymentID)
}

// IsActiveDeployment checks if the error is an active deployment.
func IsActiveDeployment(err error) bool {
	return errors.Cause(err) == ErrActiveDeployment
}

// IsNoDeployments checks if the error is no deployments.
func IsNoDeployments(err error) bool {
	return errors.Cause(err) == ErrNoDeployments
}
