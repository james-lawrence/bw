package torrentx

import "github.com/james-lawrence/torrent"

// NewMultibind bind each socket to the torrent client.
func NewMultibind(binds ...torrent.Binder) Multibind {
	return Multibind(binds)
}

// Multibind each binder to a torrent client.
type Multibind []torrent.Binder

// Bind ...
func (t Multibind) Bind(cl *torrent.Client, err error) (*torrent.Client, error) {
	for _, b := range t {
		if cl, err = b.Bind(cl, err); err != nil {
			return cl, err
		}
	}

	return cl, nil
}
