package notary

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

func newFile(path string) (s *file, err error) {
	var (
		w   *fsnotify.Watcher
		tmp *os.File
	)

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

	if err = w.Add(path); err != nil {
		return s, errors.Wrap(err, "failed to watch")
	}

	return (&file{
		w:       w,
		source:  path,
		storage: NewMem(),
	}).background(), nil
}

// file watches a file for changes.
type file struct {
	storage
	source string // path to source file.
	w      *fsnotify.Watcher
}

func (t *file) background() *file {
	ts := time.Now()
	log.Printf("authorization load initiated %s\n", t.source)
	if err := loadAuthorizedKeys(t.storage, t.source); err != nil {
		log.Println("failed to load keys", err)
	}
	log.Printf("authorization load completed %s %s\n", t.source, time.Now().Sub(ts))

	go func() {
		for {
			select {
			case evt := <-t.w.Events:
				var (
					err error
				)

				if evt.Op == fsnotify.Chmod {
					continue
				}

				log.Println("change detected", t.source, evt.Op)
				m := NewMem()
				if err = loadAuthorizedKeys(m, t.source); err != nil {
					log.Println("failed to load keys", err)
					continue
				}

				t.storage = m
			case err := <-t.w.Errors:
				log.Println("watch error", err)
			}
		}
	}()

	return t
}
