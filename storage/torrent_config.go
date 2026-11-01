package storage

import (
	"crypto/sha256"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/langx"
	"github.com/james-lawrence/torrent"
	"github.com/james-lawrence/torrent/connections"
	"github.com/james-lawrence/torrent/dht/int160"
	"github.com/james-lawrence/torrent/metainfo"
	tstorage "github.com/james-lawrence/torrent/storage"
	"github.com/pkg/errors"
)

// TorrentOption ...
type TorrentOption func(*TorrentConfig)

// TorrentOptionBind set listener binder for the torrent.
func TorrentOptionBind(b torrent.Binder) TorrentOption {
	return func(c *TorrentConfig) {
		c.b = b
	}
}

// TorrentOptionDataDir set the directory to store data in.
func TorrentOptionDataDir(a string) TorrentOption {
	return func(c *TorrentConfig) {
		c.dir = a
	}
}

// uploads are single file torrents stored directly within the data directory
// using the name of the file (the deployment id).
func dataPathMaker(baseDir string, _ int160.T, info *metainfo.Info, _ *metainfo.FileInfo) string {
	return filepath.Join(baseDir, info.Name)
}

// NewTorrent ...
func NewTorrent(cls cluster, options ...TorrentOption) (c TorrentConfig, err error) {
	var (
		util TorrentUtil
	)

	c = langx.Clone(TorrentConfig{
		c: cls,
	}, options...)

	logger := log.New(io.Discard, "", 0)
	debug := log.New(io.Discard, "", 0)
	if envx.Boolean(false, bw.EnvLogsVerbose) {
		logger = log.New(os.Stderr, "TORRENT ", 0)
		debug = log.New(os.Stderr, "TORRENT DEBUG ", 0)
	}

	// the metadata cache lives in a sub directory, which is ignored when loading the data directory.
	c.ClientConfig = torrent.NewDefaultClientConfig(
		torrent.NewMetadataCache(filepath.Join(c.dir, ".metadata")),
		tstorage.NewFile(c.dir, tstorage.FileOptionPathMaker(dataPathMaker)),
		torrent.ClientConfigInfoLogger(logger),
		torrent.ClientConfigDebugLogger(debug),
		torrent.ClientConfigSeed(true),
		torrent.ClientConfigFirewall(connections.NewFirewall(
			connections.BanInvalidPort{},
			// connections.NewBloomBanIP(10*time.Minute),
		)),
	)

	if c.client, err = c.b.Bind(torrent.NewClient(c.ClientConfig)); err != nil {
		return c, errors.WithStack(err)
	}

	if err = util.loadDir(c.dir, c.client); err != nil {
		return c, errors.WithStack(err)
	}

	return c, nil
}

// TorrentConfig ...
type TorrentConfig struct {
	b   torrent.Binder
	c   cluster
	dir string
	*torrent.ClientConfig
	client *torrent.Client
}

// Downloader ...
func (t TorrentConfig) Downloader() DownloadProtocol {
	return torrentP{
		c:      t.c,
		dir:    t.dir,
		client: t.client,
	}
}

// Uploader ...
func (t TorrentConfig) Uploader() UploadProtocol {
	return torrentP{
		c:      t.c,
		dir:    t.dir,
		client: t.client,
	}
}

type torrentP struct {
	c      cluster
	dir    string
	client *torrent.Client
	util   TorrentUtil
}

func (t torrentP) Protocol() string {
	return torrentProtocol
}

func (t torrentP) New() Downloader {
	return torrentD{
		c:      t.c,
		dir:    t.dir,
		client: t.client,
	}
}

func (t torrentP) NewUpload(bytes uint64) (Uploader, error) {
	var (
		err error
		dst *os.File
	)

	if dst, err = t.util.createFile(t.dir); err != nil {
		return nil, errors.WithStack(err)
	}

	return torrentU{
		sha:    sha256.New(),
		dst:    dst,
		client: t.client,
	}, nil
}
