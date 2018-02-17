package storage

import (
	"crypto/sha256"
	"net"
	"os"
	"time"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/torrent"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
)

// TorrentOption ...
type TorrentOption func(*TorrentConfig)

// TorrentOptionDebug debug the torrent server.
func TorrentOptionDebug(a bool) TorrentOption {
	return func(c *TorrentConfig) {
		c.Config.Debug = a
	}
}

// TorrentOptionBind set address to bind to.
func TorrentOptionBind(a net.Addr) TorrentOption {
	return func(c *TorrentConfig) {
		c.Config.ListenAddr = a.String()
		c.Config.DHTConfig.NodeId = dht.MakeDeterministicNodeID(a)
	}
}

// TorrentOptionDataDir set the directory to store data in.
func TorrentOptionDataDir(a string) TorrentOption {
	return func(c *TorrentConfig) {
		c.Config.DataDir = a
	}
}

// TorrentOptionDHTPeers set the directory to store data in.
func TorrentOptionDHTPeers(a cluster) TorrentOption {
	return func(c *TorrentConfig) {
		c.Config.DHTConfig.StartingNodes = func() (peers []dht.Addr, err error) {
			for _, peer := range a.Quorum() {
				addr := net.UDPAddr{IP: net.ParseIP(peer.Ip), Port: int(peer.TorrentPort)}
				peers = append(peers, dht.NewAddr(&addr))
			}
			return peers, err
		}
	}
}

// NewTorrent ...
func NewTorrent(options ...TorrentOption) (c TorrentConfig, err error) {
	var (
		util TorrentUtil
	)

	c = TorrentConfig{
		Config: torrent.Config{
			EncryptionPolicy: torrent.EncryptionPolicy{
				ForceEncryption: true,
			},
			Seed:              true,
			DisableTrackers:   true,
			DisableTCP:        true,
			HandshakesTimeout: 4 * time.Second,
			// HalfOpenConnsPerTorrent:    1,
			// EstablishedConnsPerTorrent: 5,
			// TorrentPeersHighWater:      5,
			// TorrentPeersLowWater:       1,
			// DHTConfig:                  util.debugDHT(),
		},
	}

	for _, opt := range options {
		opt(&c)
	}

	if c.client, err = torrent.NewClient(&c.Config); err != nil {
		return c, errors.WithStack(err)
	}

	if err = util.loadDir(c.Config.DataDir, c.client); err != nil {
		return c, errors.WithStack(err)
	}

	return c, nil
}

// TorrentConfig ...
type TorrentConfig struct {
	torrent.Config
	client *torrent.Client
}

// Downloader ...
func (t TorrentConfig) Downloader() DownloadProtocol {
	return torrentP{
		config: t.Config,
		client: t.client,
	}
}

// Uploader ...
func (t TorrentConfig) Uploader() UploadProtocol {
	return torrentP{
		config: t.Config,
		client: t.client,
	}
}

type torrentP struct {
	config torrent.Config
	client *torrent.Client
	util   TorrentUtil
}

func (t torrentP) Protocol() string {
	return torrentProtocol
}

func (t torrentP) New() Downloader {
	return torrentD{
		dir:    t.config.DataDir,
		client: t.client,
	}
}

func (t torrentP) NewUpload(uid []byte, bytes uint64) (Uploader, error) {
	var (
		err error
		dst *os.File
	)

	id := bw.RandomID(uid)
	fpath := t.util.filePath(t.config.DataDir, id.String())

	if dst, err = t.util.createFile(fpath); err != nil {
		return nil, errors.WithStack(err)
	}

	return torrentU{
		uid:    id,
		sha:    sha256.New(),
		dst:    dst,
		client: t.client,
	}, nil
}
