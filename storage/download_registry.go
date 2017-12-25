package storage

import (
	"log"
	"strings"

	"github.com/pkg/errors"
)

const (
	protocolSuffix = "://"
)

type protocol interface {
	Protocol() string
	New(location string) Downloader
}

// Option for a download registry.
type Option func(*Registry)

// OptionProtocols set the protocols available for this registry.
func OptionProtocols(protocols ...protocol) Option {
	pp := make(map[string]protocol, len(protocols))
	for _, p := range protocols {
		pp[p.Protocol()] = p
	}

	return func(r *Registry) {
		r.protocols = pp
	}
}

// New create a new downloader registry.
func New(options ...Option) Registry {
	r := Registry{}

	for _, opt := range options {
		opt(&r)
	}

	return r
}

// Registry of protocols for downloads.
type Registry struct {
	protocols map[string]protocol
}

// New connect to the specified location by creating a io.ReadCloser
func (t Registry) New(location string) Downloader {
	if strings.HasPrefix(location, fileProtocol) {
		log.Println("downloading from local file")
		return DownloadFile{Path: strings.TrimPrefix(location, fileProtocol+"://")}
	}

	for prefix, p := range t.protocols {
		if strings.HasPrefix(location, prefix) {
			return p.New(location)
		}
	}

	return downloader{newErrReader(errors.Errorf("unknown protocol: [%s]", location))}
}
