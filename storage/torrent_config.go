package storage

import (
	"crypto/sha256"
	"log"
	"net"
	"os"
	"time"
	// "math"

	"github.com/anacrolix/dht"
	// "github.com/anacrolix/dht/krpc"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	// "github.com/anacrolix/torrent/tracker"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
)

// TorrentOption ...
type TorrentOption func(*TorrentConfig)

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
			for _, peer := range a.Peers() {
				if a.Local().Name == peer.Name {
					continue
				}

				addr := net.UDPAddr{IP: net.ParseIP(peer.Ip), Port: int(peer.TorrentPort)}
				log.Println("ADDED DHT PEER", peer.Name, addr.String())
				peers = append(peers, dht.NewAddr(&addr))
			}
			return peers, err
		}
	}
}

// NewTorrent ...
func NewTorrent(options ...TorrentOption) (c TorrentConfig, err error) {
	c = TorrentConfig{
		Config: torrent.Config{
			// Debug:           true,
			Seed:            true,
			DisableTrackers: true,
			DHTConfig:       dht.ServerConfig{},
			// DHTConfig: torrentUtil{}.debugDHT(),
		},
	}

	for _, opt := range options {
		opt(&c)
	}

	if c.client, err = torrent.NewClient(&c.Config); err != nil {
		return c, errors.WithStack(err)
	}

	mi := metainfo.MetaInfo{}
	mi.SetDefaults()
	if c.announce, err = c.client.DHT().Announce(mi.HashInfoBytes(), 0, true); err != nil {
		return c, errors.WithStack(err)
	}

	go func() {
		for _ = range time.Tick(30 * time.Second) {
			c.client.DHT().WriteStatus(os.Stderr)
		}
	}()
	return c, nil
}

// TorrentConfig ...
type TorrentConfig struct {
	torrent.Config
	client   *torrent.Client
	announce *dht.Announce
}

// Downloader ...
func (t TorrentConfig) Downloader() DownloadProtocol {
	return torrentP{
		config: t.Config,
		client: t.client,
	}
}

// Uploader ...
func (t TorrentConfig) Uploader() (_ Protocol) {
	return torrentP{
		config: t.Config,
		client: t.client,
	}
}

type torrentP struct {
	config torrent.Config
	client *torrent.Client
	util   torrentUtil
}

func (t torrentP) Protocol() string {
	return torrentProtocol
}

func (t torrentP) New(location string) Downloader {
	return torrentD{
		dir:    t.config.DataDir,
		magnet: location,
		client: t.client,
	}
}

func (t torrentP) NewUpload(uid []byte, bytes uint64) (Uploader, error) {
	var (
		err error
		dst *os.File
	)

	id := bw.RandomID(uid)
	fpath := t.util.FilePath(t.config.DataDir, id.String())

	if dst, err = t.util.CreateFile(fpath); err != nil {
		return nil, errors.WithStack(err)
	}

	return torrentU{
		uid:    id,
		sha:    sha256.New(),
		dst:    dst,
		client: t.client,
	}, nil
}