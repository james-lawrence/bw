package certificatecache

import (
	"crypto/tls"
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/james-lawrence/bw"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
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
func NewDirectory(serverName, dir string) (cache Directory) {
	w := mustWatcher(dir)
	return Directory{
		serverName: serverName,
		dir:        dir,
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
	cachedCert *tls.Certificate
	watcher    *fsnotify.Watcher
	initialize *sync.Once
	m          *sync.Mutex
}

func (t Directory) init() (err error) {
	t.initialize.Do(func() {
		err = t.refresh()
		go t.background()
	})

	return errors.Wrap(err, "failed to load tls credentials")
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
	limit := rate.NewLimiter(rate.Every(time.Second), 1)
	for {
		select {
		case _ = <-t.watcher.Events:
			if limit.Allow() {
				log.Println("refreshing certificates")
			}
			t.refresh()
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
	log.Println("loading", certpath, keypath)

	if cert, err = tls.LoadX509KeyPair(certpath, keypath); err != nil {
		return errors.WithStack(err)
	}

	t.m.Lock()
	*t.cachedCert = cert
	t.m.Unlock()

	return nil
}
