package storage

import (
	"crypto/sha256"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/torrent"
	"github.com/james-lawrence/torrent/connections"
	"github.com/james-lawrence/torrent/dht/v2"
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
		c.ClientConfig.DataDir = a
	}
}

// TorrentOptionDHTPeers set the directory to store data in.
func TorrentOptionDHTPeers(a cluster) TorrentOption {
	return func(c *TorrentConfig) {
		c.ClientConfig.DhtStartingNodes = func() (peers []dht.Addr, err error) {
			for _, peer := range a.Quorum() {
				addr := net.UDPAddr{IP: net.ParseIP(peer.Ip), Port: int(peer.TorrentPort)}
				peers = append(peers, dht.NewAddr(&addr))
			}
			return peers, err
		}
	}
}

// NewTorrent ...
func NewTorrent(cls cluster, options ...TorrentOption) (c TorrentConfig, err error) {
	var (
		util TorrentUtil
	)

	c = TorrentConfig{
		c:            cls,
		ClientConfig: torrent.NewDefaultClientConfig(),
	}

	c.ClientConfig.Logger = log.New(ioutil.Discard, "", 0)
	if envx.Boolean(false, bw.EnvLogsVerbose) {
		c.ClientConfig.Logger = log.New(os.Stderr, "TORRENT ", 0)
		c.ClientConfig.Debug = log.New(os.Stderr, "TORRENT DEBUG ", 0)
	}

	c.ClientConfig.Seed = true
	c.ClientConfig.NoDefaultPortForwarding = true
	c.ClientConfig.Handshaker = connections.NewHandshaker(
		connections.NewFirewall(
			connections.BanInvalidPort{},
			// connections.NewBloomBanIP(10*time.Minute),
		),
	)

	for _, opt := range options {
		opt(&c)
	}

	if c.client, err = c.b.Bind(torrent.NewClient(c.ClientConfig)); err != nil {
		return c, errors.WithStack(err)
	}

	if err = util.loadDir(c.ClientConfig.DataDir, c.client); err != nil {
		return c, errors.WithStack(err)
	}

	go util.Bootstrap(c.client)

	return c, nil
}

// TorrentConfig ...
type TorrentConfig struct {
	b torrent.Binder
	c cluster
	*torrent.ClientConfig
	client *torrent.Client
}

// Downloader ...
func (t TorrentConfig) Downloader() DownloadProtocol {
	return torrentP{
		c:      t.c,
		config: t.ClientConfig,
		client: t.client,
	}
}

// Uploader ...
func (t TorrentConfig) Uploader() UploadProtocol {
	return torrentP{
		c:      t.c,
		config: t.ClientConfig,
		client: t.client,
	}
}

type torrentP struct {
	c      cluster
	config *torrent.ClientConfig
	client *torrent.Client
	util   TorrentUtil
}

func (t torrentP) Protocol() string {
	return torrentProtocol
}

func (t torrentP) New() Downloader {
	return torrentD{
		c:      t.c,
		dir:    t.config.DataDir,
		client: t.client,
	}
}

func (t torrentP) NewUpload(bytes uint64) (Uploader, error) {
	var (
		err error
		dst *os.File
	)

	if dst, err = t.util.createFile(t.config.DataDir); err != nil {
		return nil, errors.WithStack(err)
	}

	return torrentU{
		sha:    sha256.New(),
		dst:    dst,
		client: t.client,
	}, nil
}
