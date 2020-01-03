package notary

import (
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

func newFile(path string) (s *file, err error) {
	var (
		w   *fsnotify.Watcher
		tmp *os.File
	)

	// ensure the file we are watching exists
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
		storage: newMem(),
	}).background(), nil
}

// file watches a file for changes.
type file struct {
	storage
	source string // path to source file.
	w      *fsnotify.Watcher
}

func (t *file) background() *file {
	log.Println("loading", t.source)
	if err := loadAuthorizedKeys(t.storage, t.source); err != nil {
		log.Println("failed to load keys", err)
	}

	go func() {
		for {
			select {
			case evt := <-t.w.Events:
				var (
					err error
				)
				log.Println("change detected", t.source, evt.Op)
				m := newMem()
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
