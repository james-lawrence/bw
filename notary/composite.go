package notary

import (
	"context"
	"log"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/errorsx"
)

// NewComposite storage.
func NewComposite(root string, p storage, buckets ...storage) Composite {
	return Composite{
		Root:    root,
		primary: p,
		buckets: buckets,
	}
}

// Composite combine multiple storages into a single api.
type Composite struct {
	Root    string    // root of the authorization directory.
	primary storage   // the mutatable bucket, allows for insertions.
	buckets []storage // read only buckets. cannot be mutated by the composite.
}

// Lookup scan each bucket for the fingerprint starting with the primary.
// returns the last error encountered.
func (t Composite) Lookup(fingerprint string) (g *Grant, err error) {
	if g, err = t.primary.Lookup(fingerprint); err == nil {
		return g, err
	}

	for _, b := range t.buckets {
		if g, err = b.Lookup(fingerprint); err == nil {
			return g, err
		}
	}

	return g, err
}

// Bloomfilter generates a bloom filter representing the current state of the notary system.
func (t Composite) Bloomfilter(ctx context.Context) (b *bloom.BloomFilter, err error) {
	var (
		c    chan *Grant = make(chan *Grant)
		errc chan error  = make(chan error, 1)
	)

	log.Println("generating bloomfilter initiated")
	defer log.Println("generating bloomfilter completed")

	b = bloom.NewWithEstimates(1000, 0.0001)

	go func() {
		defer close(c)
		if err = t.Sync(ctx, b, c); err != nil {
			errc <- err
		}
	}()

	for g := range c {
		debugx.Println("bloomfilter adding", g.Fingerprint)
		b.AddString(g.Fingerprint)
	}

	select {
	case err = <-errc:
		return b, err
	default:
		return b, nil
	}
}

// Sync find grants not in the bloom filter and insert them into the channel.
// because we're syncing we handle errors loosely. every bucket will be attempted
// before returning the first error encountered.
func (t Composite) Sync(ctx context.Context, b Bloomy, c chan *Grant) (err error) {
	err = t.sync(ctx, t.primary, b, c)

	for _, s := range t.buckets {
		err = errorsx.Compact(err, t.sync(ctx, s, b, c))
	}

	return err
}

func (t Composite) sync(ctx context.Context, s storage, b Bloomy, c chan *Grant) error {
	var (
		ok bool
		ss SyncStorage
	)

	if ss, ok = s.(SyncStorage); !ok {
		return nil
	}

	return ss.Sync(ctx, b, c)
}

// Insert the grant into the primary.
func (t Composite) Insert(g *Grant) (*Grant, error) {
	return t.primary.Insert(g)
}

// Delete the grant from the primary.
func (t Composite) Delete(g *Grant) (*Grant, error) {
	return t.primary.Delete(g)
}
