package observers

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewDirectory see Directory
func NewDirectory(dir string) (cache Directory, err error) {
	var (
		w *fsnotify.Watcher
	)

	if w, err = fsnotify.NewWatcher(); err != nil {
		return cache, err
	}

	if err = w.Add(dir); err != nil {
		return cache, err
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if path == dir {
			return nil
		}

		return os.RemoveAll(path)
	})

	if err != nil {
		return cache, err
	}

	cache = Directory{
		dir:        dir,
		watcher:    w,
		initialize: &sync.Once{},
		m:          &sync.RWMutex{},
		observers:  map[string]Conn{},
	}

	go cache.background()

	return cache, nil
}

// Directory observes a directory for sockets to write messages into.
type Directory struct {
	dir        string
	watcher    *fsnotify.Watcher
	initialize *sync.Once
	m          *sync.RWMutex
	observers  map[string]Conn
}

// Observers number of current observers
func (t Directory) Observers() int {
	t.m.RLock()
	defer t.m.RUnlock()
	return len(t.observers)
}

// Connect ...
func (t Directory) Connect(b chan agent.Message) (l net.Listener, s *grpc.Server, err error) {
	var (
		id bw.RandomID
	)

	if id, err = bw.SimpleGenerateID(); err != nil {
		return l, s, err
	}

	addr := filepath.Join(t.dir, fmt.Sprintf("%s.sock", id.String()))
	if l, err = net.Listen("unix", addr); err != nil {
		return l, s, err
	}

	s = New(b)
	go s.Serve(l)

	return l, s, nil
}

// Dispatch messages to the observers.
func (t Directory) Dispatch(ctx context.Context, messages ...agent.Message) error {
	t.m.RLock()
	cpy := make([]Conn, 0, len(t.observers))
	for _, obs := range t.observers {
		cpy = append(cpy, obs)
	}
	t.m.RUnlock()

	if len(cpy) == 0 {
		return nil
	}

	log.Println("observer dispatch initiated", len(cpy))
	defer log.Println("observer dispatch completed", len(cpy))

	for _, conn := range cpy {
		if err := conn.Dispatch(ctx, messages...); err != nil {
			log.Println(errors.Wrap(err, "failed to deliver messages"))
		}
	}

	return nil
}

func (t Directory) background() {
	for {
		select {
		case e := <-t.watcher.Events:
			switch e.Op {
			case fsnotify.Create:
				logx.MaybeLog(t.connect(e))
			case fsnotify.Remove:
				logx.MaybeLog(t.disconnect(e))
			}
			log.Println("open connections", t.Observers())
		case err := <-t.watcher.Errors:
			log.Println("watch error", err)
		}
	}
}

func (t Directory) connect(e fsnotify.Event) (err error) {
	var (
		conn Conn
	)

	log.Println("new observer", e.Name)

	t.m.Lock()
	defer t.m.Unlock()
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()

	if conn, err = NewDialer(ctx, e.Name, grpc.WithInsecure(), grpc.WithBlock()); err != nil {
		return errors.Wrap(err, "failed to connect to observer")
	}

	t.observers[e.Name] = conn

	return err
}

func (t *Directory) disconnect(e fsnotify.Event) (err error) {
	var (
		ok   bool
		conn Conn
	)

	log.Println("lost observer", e.Name)

	t.m.Lock()
	defer t.m.Unlock()

	if conn, ok = t.observers[e.Name]; !ok {
		return errorsx.String("disconnect for a non-existant socket")
	}

	if err = errors.Wrap(conn.conn.Close(), "failed to disconnect"); err != nil {
		return err
	}

	delete(t.observers, e.Name)

	return err
}
