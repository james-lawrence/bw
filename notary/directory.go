package notary

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
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

func (t Directory) lookup(fingerprint string) (key string, g Grant, err error) {
	var (
		encoded []byte
	)

	if strings.TrimSpace(fingerprint) == "" {
		return key, g, errors.New("can not use an empty fingerprint")
	}

	key = genKey(t.root, fingerprint)

	if encoded, err = ioutil.ReadFile(key); err != nil {
		return key, g, errors.Wrapf(err, "unable to read %s", key)
	}

	if err = proto.Unmarshal(encoded, &g); err != nil {
		return key, g, errors.Wrapf(err, "unable to read %s", key)
	}

	return key, g, err
}

// Lookup a grant.
func (t Directory) Lookup(fingerprint string) (g Grant, err error) {
	t.m.RLock()
	defer t.m.RUnlock()

	_, g, err = t.lookup(fingerprint)

	return g, err
}

// Insert a grant
func (t Directory) Insert(g Grant) (_ Grant, err error) {
	var (
		encoded []byte
		dst     *os.File
	)

	g = g.EnsureDefaults()
	key := genKey(t.root, g.Fingerprint)

	if encoded, err = proto.Marshal(&g); err != nil {
		return g, errors.Wrapf(err, "unable to write %s", key)
	}

	t.m.Lock()
	defer t.m.Unlock()

	if err = os.MkdirAll(filepath.Dir(key), 0700); err != nil {
		return g, errors.Wrapf(err, "unable to write %s", key)
	}

	if dst, err = os.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600); err != nil {
		return g, errors.Wrapf(err, "unable to write %s", key)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, bytes.NewReader(encoded)); err != nil {
		return g, errors.Wrapf(err, "unable to write %s", key)
	}

	if err = dst.Sync(); err != nil {
		return g, errors.Wrapf(err, "unable to write %s", key)
	}

	return g, err
}

// Delete a grant
func (t Directory) Delete(g Grant) (_ Grant, err error) {
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
