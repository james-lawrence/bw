package certificatecache

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/errorsx"
)

func mustWatcher(dir string) *fsnotify.Watcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	if err = w.Add(dir); err != nil {
		panic(err)
	}

	return w
}

// NewDirectory maintains a certificate config by watching a directory.
func NewDirectory(serverName, dir string, pool *x509.CertPool) (cache Directory) {
	w := mustWatcher(dir)
	return Directory{
		serverName: serverName,
		dir:        dir,
		pooldir:    filepath.Join(dir, "authorities"),
		pool:       pool,
		watcher:    w,
		cachedCert: &tls.Certificate{}, // prevent nil exception if something goes wrong on initial load.
		initialize: &sync.Once{},
		m:          &sync.Mutex{},
	}
}

// Directory manages the certificates by watching a directory
// and reloading when necessary.
type Directory struct {
	serverName string
	dir        string
	pooldir    string
	pool       *x509.CertPool
	cachedCert *tls.Certificate
	watcher    *fsnotify.Watcher
	initialize *sync.Once
	m          *sync.Mutex
}

func (t Directory) init() (err error) {
	t.initialize.Do(func() {
		err = errorsx.Compact(
			os.RemoveAll(t.pooldir),
			os.MkdirAll(t.pooldir, 0700),
			t.refresh(),
		)

		if err == nil {
			go t.background()
		}
	})

	return errors.Wrap(err, "failed to initialize certificate cache")
}

// InsertAuthority insert the cluster authority into the pool directory.
func (t Directory) InsertAuthority(pemCerts []byte) (err error) {
	if err = t.init(); err != nil {
		return err
	}

	return nil
}

// GetCertificate for use by tls.Config.
func (t Directory) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return t.cert()
}

// GetClientCertificate for use by tls.Config.
func (t Directory) GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return t.cert()
}

func (t Directory) background() {
	limit := rate.NewLimiter(rate.Every(10*time.Second), 1)
	debounce := make(chan struct{})
	go func() {
		for _ = range debounce {
			if err := limit.Wait(context.Background()); err != nil {
				log.Println("debounce wait failed", err)
				continue
			}

			log.Println("refreshing certificates")
			t.refresh()
		}
	}()

	for {
		select {
		case _ = <-t.watcher.Events:
			select {
			case debounce <- struct{}{}:
			default:
			}
		case err := <-t.watcher.Errors:
			if limit.Allow() {
				log.Println("watch error", err)
			}
		}
	}
}

func (t Directory) cert() (cert *tls.Certificate, err error) {
	t.init()
	t.m.Lock()
	cert = t.cachedCert
	t.m.Unlock()

	if cert == nil {
		return nil, errors.Errorf("certificate missing in: %s", t.dir)
	}

	return cert, nil
}

func (t Directory) refresh() (err error) {
	var (
		certpath, keypath string
		cert              tls.Certificate
	)

	certpath = bw.LocateFirstInDir(t.dir, DefaultTLSCertServer, DefaultTLSCertClient)
	keypath = bw.LocateFirstInDir(t.dir, DefaultTLSKeyServer, DefaultTLSKeyClient)

	debugx.Println("loading", certpath, keypath)

	if cert, err = tls.LoadX509KeyPair(certpath, keypath); err != nil {
		return errors.WithStack(err)
	}

	t.m.Lock()
	defer t.m.Unlock()

	*t.cachedCert = cert

	// refresh the pool
	return filepath.Walk(t.pooldir, func(path string, info os.FileInfo, err error) error {
		var (
			ca []byte
		)

		if err != nil {
			log.Println("error walking authority cache", err)
			return nil
		}

		if info.IsDir() && path == t.pooldir {
			return nil
		}

		if info.IsDir() {
			return filepath.SkipDir
		}

		if ca, err = ioutil.ReadFile(path); err != nil {
			log.Println("failed to read certificate", path)
			return nil
		}

		if ok := t.pool.AppendCertsFromPEM(ca); !ok {
			return nil
		}

		return nil
	})
}
