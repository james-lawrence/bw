package notary

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// NewDirectory store credentials in the given directory.
func NewDirectory(root string) Directory {
	return Directory{root: root, m: &sync.RWMutex{}}
}

// Directory storage stores credentials within the given directory.
type Directory struct {
	root string
	m    *sync.RWMutex
}

func (t Directory) lookup(fingerprint string) (key string, g *Grant, err error) {
	if strings.TrimSpace(fingerprint) == "" {
		return key, nil, errors.New("can not use an empty fingerprint")
	}

	key = genKey(t.root, fingerprint)
	g, err = t.read(key)

	return key, g, err
}

func (t Directory) read(path string) (g *Grant, err error) {
	var (
		encoded []byte
	)

	g = &Grant{}

	if encoded, err = os.ReadFile(path); err != nil {
		return nil, errors.Wrapf(err, "unable to read %s", path)
	}

	if err = proto.Unmarshal(encoded, g); err != nil {
		return nil, errors.Wrapf(err, "unable to read %s", path)
	}

	return g, nil
}

// Lookup a grant.
func (t Directory) Lookup(fingerprint string) (g *Grant, err error) {
	t.m.RLock()
	defer t.m.RUnlock()

	_, g, err = t.lookup(fingerprint)

	return g, err
}

func (t Directory) Sync(ctx context.Context, b Bloomy, c chan *Grant) error {
	t.m.RLock()
	defer t.m.RUnlock()
	return filepath.Walk(t.root, func(path string, d os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		g, err := t.read(path)
		if err != nil {
			log.Println("unable to read", path, "skipping")
			return nil
		}

		if b.Test([]byte(g.Fingerprint)) {
			return nil
		}

		select {
		case c <- g:
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})
}

// Insert a grant
func (t Directory) Insert(g *Grant) (_ *Grant, err error) {
	var (
		encoded []byte
		dst     *os.File
	)

	gd := g.EnsureDefaults()
	key := genKey(t.root, gd.Fingerprint)

	if encoded, err = proto.Marshal(gd); err != nil {
		return nil, errors.Wrapf(err, "unable to write %s", key)
	}

	t.m.Lock()
	defer t.m.Unlock()

	if err = os.MkdirAll(filepath.Dir(key), 0700); err != nil {
		return nil, errors.Wrapf(err, "unable to write %s", key)
	}

	if dst, err = os.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600); err != nil {
		return nil, errors.Wrapf(err, "unable to write %s", key)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, bytes.NewReader(encoded)); err != nil {
		return nil, errors.Wrapf(err, "unable to write %s", key)
	}

	if err = dst.Sync(); err != nil {
		return nil, errors.Wrapf(err, "unable to write %s", key)
	}

	return gd, err
}

// Delete a grant
func (t Directory) Delete(g *Grant) (_ *Grant, err error) {
	var (
		key string
	)

	if key, g, err = t.lookup(g.Fingerprint); err != nil {
		return g, err
	}

	t.m.Lock()
	defer t.m.Unlock()

	if err := os.Remove(key); err != nil {
		return g, err
	}

	return g, nil
}
