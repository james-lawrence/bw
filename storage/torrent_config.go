package storage

import (
	"crypto/sha256"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/james-lawrence/torrent/dht/v2"
	"github.com/james-lawrence/torrent"
	"github.com/pkg/errors"
)

// TorrentOption ...
type TorrentOption func(*TorrentConfig)

// TorrentOptionBind set address to bind to.
func TorrentOptionBind(a net.Addr) TorrentOption {
	return func(c *TorrentConfig) {
		c.addr = a
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
	c.ClientConfig.DisableIPv6 = true
	c.ClientConfig.Logger = log.New(ioutil.Discard, "", 0)
	c.ClientConfig.Seed = true

	for _, opt := range options {
		opt(&c)
	}

	autobind := torrent.NewAutobindSpecified(c.addr.String())
	if c.client, err = autobind.Bind(torrent.NewClient(c.ClientConfig)); err != nil {
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
	addr net.Addr
	c    cluster
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
