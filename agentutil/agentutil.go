// Package agentutil provides common utilities for an agent
// Examples include removing old files from directories etc.
package agentutil

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"google.golang.org/grpc"

	agentx "bitbucket.org/jatone/bearded-wookie/agent"
	clusterp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

// DialPeer dial the peer
func DialPeer(p agent.Peer, options ...grpc.DialOption) (zeroc agentx.Client, err error) {
	var (
		addr string
	)

	if addr = RPCAddress(p); addr == "" {
		return zeroc, errors.Errorf("failed to determine address of peer: %s", p.Name)
	}

	return agentx.DialClient(addr, options...)
}

type client interface {
	Upload(srcbytes uint64, src io.Reader) (agent.Archive, error)
	Deploy(info agent.Archive) error
	Credentials() (agent.Peer, []string, []byte, error)
	Info() (agent.Status, error)
	Watch(out chan<- agent.Message) error
	Dispatch(messages ...agent.Message) error
	Close() error
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
		return i.Info.ModTime().After(j.Info.ModTime())
	})
}

// RPCAddress for peer.
func RPCAddress(p agent.Peer) string {
	return clusterp.RPCAddress(p)
}

// NodeRPCAddress returns the node's rpc address.
// if an error occurs it returns a blank string.
func NodeRPCAddress(n *memberlist.Node) string {
	return clusterp.NodeRPCAddress(n)
}
