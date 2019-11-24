package notary

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/james-lawrence/bw"
	"github.com/pkg/errors"
)

// NewFromFile load notary configuration from a file.
func NewFromFile(path string) (c Storage, err error) {
	var (
		in io.ReadCloser
	)

	if in, err = os.Open(path); err != nil {
		return c, errors.Wrapf(err, "failed to read configuration from: %s", path)
	}
	defer in.Close()

	if c, err = NewFrom(in); err != nil {
		return c, errors.Wrapf(err, "failed to read configuration from: %s", path)
	}

	return c, nil
}

// NewFrom parse the configuration from an io.Reader.
func NewFrom(in io.Reader) (c Storage, err error) {
	var (
		nc  Storage
		bin []byte
	)

	if bin, err = ioutil.ReadAll(in); err != nil {
		return c, err
	}

	if err = bw.ExpandAndDecode(bin, &nc); err != nil {
		return c, err
	}

	return nc, nil
}

// NewStorage storage.
func NewStorage(root string) Storage {
	return Storage{
		root: root,
		m:    &sync.RWMutex{},
	}
}

type nconfig struct {
	AuthorizedKeys string `yaml:"authorizedKeys"`
}

// Storage ...
type Storage struct {
	root   string  `yaml:"root"`
	config nconfig `yaml:"notary"`
	m      *sync.RWMutex
}

func (t Storage) lookup(fingerprint string) (key string, g Grant, err error) {
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
func (t Storage) Lookup(fingerprint string) (g Grant, err error) {
	t.m.RLock()
	defer t.m.RUnlock()

	_, g, err = t.lookup(fingerprint)
	return g, err
}

// Insert a grant
func (t Storage) Insert(g Grant) (_ Grant, err error) {
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
func (t Storage) Delete(g Grant) (_ Grant, err error) {
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

func genKey(root, fingerprint string) string {
	return filepath.Join(root, bw.DirNotary, fingerprint)
}

func genFingerprint(d []byte) string {
	digest := sha256.Sum256(d)
	return hex.EncodeToString(digest[:])
}
