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
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/systemx"
)

func mustWatcher(dir string) *fsnotify.Watcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	return w
}

// NewDirectory maintains a certificate config by watching a directory.
func NewDirectory(serverName, dir, ca string, pool *x509.CertPool) (cache Directory) {
	w := mustWatcher(dir)
	d := Directory{
		serverName: serverName,
		caFile:     ca,
		dir:        dir,
		pooldir:    filepath.Join(dir, "authorities"),
		pool:       pool,
		watcher:    w,
		cachedCert: &tls.Certificate{},
		initialize: &sync.Once{},
		m:          &sync.Mutex{},
	}

	// this is necessary to initialize the clients with the correct CAs
	logx.MaybeLog(d.init())

	return d
}

// Directory manages the certificates by watching a directory
// and reloading when necessary.
type Directory struct {
	serverName string
	caFile     string
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
		err = logx.MaybeLog(errorsx.Compact(
			os.MkdirAll(t.pooldir, 0700),
			t.watcher.Add(t.dir),
			t.watcher.Add(t.pooldir),
			t.refresh(),
		))

		if err == nil {
			go t.background()
		}
	})

	return errors.Wrap(err, "failed to initialize certificate cache")
}

// GetCertificate for use by tls.Config.
func (t Directory) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return t.cert()
}

// GetClientCertificate for use by tls.Config.
func (t Directory) GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return t.cert()
}

func (t Directory) background() {
	limit := rate.NewLimiter(rate.Every(10*time.Second), 2)
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
	if err = t.init(); err != nil {
		return nil, err
	}

	t.m.Lock()
	cert = t.cachedCert
	t.m.Unlock()

	if cert == nil {
		return nil, logx.MaybeLog(errors.Errorf("certificate missing in: %s", t.dir))
	}

	return cert, nil
}

func (t Directory) load(path string) (err error) {
	var ca []byte

	if ca, err = ioutil.ReadFile(path); err != nil {
		return errors.Wrapf(err, "failed to read certificate: %s", path)
	}

	if ok := t.pool.AppendCertsFromPEM(ca); !ok {
		return nil
	}

	return nil
}

func (t Directory) refresh() (err error) {
	var (
		certpath, keypath string
		cert              tls.Certificate
	)

	certpath = bw.LocateFirstInDir(t.dir, DefaultTLSCertServer, DefaultTLSBootstrapCert)
	keypath = bw.LocateFirstInDir(t.dir, DefaultTLSKeyServer)

	debugx.Println("loading", certpath, keypath)

	if cert, err = tls.LoadX509KeyPair(certpath, keypath); err != nil {
		return errors.WithStack(err)
	}

	t.m.Lock()
	defer t.m.Unlock()

	*t.cachedCert = cert

	if systemx.FileExists(t.caFile) {
		if err = t.load(t.caFile); err != nil {
			return err
		}
	}

	// refresh the pool
	return filepath.Walk(t.pooldir, func(path string, info os.FileInfo, err error) error {
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

		if err = t.load(path); err != nil {
			log.Println(err)
			return nil
		}

		return nil
	})
}
