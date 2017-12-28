package storage

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Downloader ...
type Downloader interface {
	Download() io.ReadCloser
}

func newDownload(rc io.ReadCloser) rcDownload {
	return rcDownload{rc}
}

type rcDownload struct {
	io.ReadCloser
}

func (t rcDownload) Download() io.ReadCloser {
	return t.ReadCloser
}

// DownloadProtocol ...
type DownloadProtocol interface {
	Protocol() string
	New(location string) Downloader
}

type downloadConfig interface {
	Downloader() (DownloadProtocol, error)
}

// Option for a download registry.
type Option func(*Registry)

// OptionProtocols set the protocols available for this registry.
func OptionProtocols(protocols ...DownloadProtocol) Option {
	pp := make(map[string]DownloadProtocol, len(protocols))
	for _, p := range protocols {
		pp[p.Protocol()] = p
	}

	return func(r *Registry) {
		r.protocols = pp
	}
}

// OptionDefaultProtocols checks for well known configuration files in the given directory.
// well known files:
// s3.storage
func OptionDefaultProtocols(dir string, protocols ...DownloadProtocol) Option {
	var (
		err error
		p   DownloadProtocol
	)

	if p, err = loadDownloadFromFile(filepath.Join(dir, "s3.storage"), defaultS3Config); err != nil {
		log.Println("failed to load s3 download configuration", err)
	} else {
		protocols = append(protocols, p)
	}

	return OptionProtocols(protocols...)
}

func loadDownloadFromFile(path string, p downloadConfig) (_ DownloadProtocol, err error) {
	var (
		b []byte
	)

	if b, err = ioutil.ReadFile(path); err != nil && !os.IsNotExist(err) {
		return nil, errors.WithStack(err)
	}

	return newDownloadProtocolFromConfig(b, p)
}

func newDownloadProtocolFromConfig(serialized []byte, v downloadConfig) (_ DownloadProtocol, err error) {
	if err = errors.WithStack(yaml.Unmarshal(serialized, v)); err != nil {
		return nil, err
	}

	return v.Downloader()
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
	protocols map[string]DownloadProtocol
}

// New connect to the specified location by creating a io.ReadCloser
func (t Registry) New(location string) Downloader {
	for prefix, p := range t.protocols {
		if strings.HasPrefix(location, prefix) {
			return p.New(location)
		}
	}

	return newDownload(newErrReader(errors.Errorf("unknown protocol: [%s]", location)))
}
