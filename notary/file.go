package notary

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

func newFile(path string) (s *file, err error) {
	var (
		w   *fsnotify.Watcher
		tmp *os.File
	)

	path = filepath.Clean(path)

	// ensure the file we are watching exists
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return s, err
	}

	if tmp, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err != nil {
		return s, err
	}
	defer tmp.Close()

	if w, err = fsnotify.NewWatcher(); err != nil {
		return s, err
	}

	// watch the directory rather than the file; the file is replaced atomically
	// via rename which would otherwise invalidate the watch.
	if err = w.Add(filepath.Dir(path)); err != nil {
		return s, errors.Wrap(err, "failed to watch")
	}

	return (&file{
		w:       w,
		source:  path,
		m:       &sync.RWMutex{},
		storage: NewMem(),
	}).background(), nil
}

// file watches a file for changes.
type file struct {
	source  string // path to source file.
	w       *fsnotify.Watcher
	m       *sync.RWMutex
	storage storage
}

func (t *file) current() storage {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.storage
}

func (t *file) Lookup(fingerprint string) (*Grant, error) {
	return t.current().Lookup(fingerprint)
}

func (t *file) Insert(g *Grant) (*Grant, error) {
	return t.current().Insert(g)
}

func (t *file) Delete(g *Grant) (*Grant, error) {
	return t.current().Delete(g)
}

func (t *file) Sync(ctx context.Context, b Bloomy, c chan *Grant) error {
	return t.current().Sync(ctx, b, c)
}

// reload the grants from the source file.
func (t *file) reload() error {
	m := NewMem()
	if err := LoadAuthorizedKeys(m, t.source); err != nil {
		return err
	}

	t.m.Lock()
	t.storage = m
	t.m.Unlock()

	return nil
}

func (t *file) background() *file {
	ts := time.Now()
	log.Printf("authorization load initiated %s\n", t.source)
	if err := t.reload(); err != nil {
		log.Println("failed to load keys", err)
	}
	log.Printf("authorization load completed %s %s\n", t.source, time.Since(ts))

	go func() {
		for {
			select {
			case evt := <-t.w.Events:
				if filepath.Clean(evt.Name) != t.source {
					continue
				}

				if !evt.Has(fsnotify.Create) && !evt.Has(fsnotify.Write) {
					continue
				}

				log.Println("change detected", t.source, evt.Op)
				if err := t.reload(); err != nil {
					log.Println("failed to load keys", err)
				}
			case err := <-t.w.Errors:
				log.Println("watch error", err)
			}
		}
	}()

	return t
}
