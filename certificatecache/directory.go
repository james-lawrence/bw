package certificatecache

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/systemx"
)

func mustWatcher() *fsnotify.Watcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	return w
}

// NewDirectory maintains a certificate config by watching a directory.
func NewDirectory(serverName, dir, ca string, pool *x509.CertPool) (cache *Directory) {
	log.Println("NewDirectory cert cache", serverName, dir, ca)
	w := mustWatcher()
	d := &Directory{
		serverName: serverName,
		caFile:     ca,
		dir:        dir,
		pooldir:    filepath.Join(dir, "authorities"),
		pool:       pool,
		watcher:    w,
		initialize: &sync.Once{},
		m:          &sync.Mutex{},
	}

	// this is necessary to initialize the clients with the correct CAs
	errorsx.MaybeLog(d.init())

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

func (t *Directory) init() (err error) {
	t.initialize.Do(func() {
		err = errorsx.Compact(
			errors.Wrap(os.MkdirAll(t.pooldir, 0700), "failed to create authority directory"),
			errors.Wrap(t.watcher.Add(t.dir), "failed to watch tls directory"),
			errors.Wrap(t.watcher.Add(t.pooldir), "failed to watch authority directory"),
			errors.Wrap(t.refresh(), "failed to refresh"),
		)
		go t.background()
	})

	err = errors.Wrap(err, "failed to initialize certificate cache")
	errorsx.MaybeLog(err)
	return err
}

// GetCertificate for use by tls.Config.
func (t *Directory) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return t.cert()
}

// GetClientCertificate for use by tls.Config.
func (t *Directory) GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return t.cert()
}

func (t *Directory) background() {
	debounce := time.NewTimer(time.Second)
	limit := rate.NewLimiter(rate.Every(10*time.Second), 2)
	for {
		select {
		case <-debounce.C:
			errorsx.MaybeLog(t.refresh())
		case <-t.watcher.Events:
			debounce.Reset(time.Second)
		case err := <-t.watcher.Errors:
			if limit.Allow() {
				log.Println("watch error", err)
			}
		}
	}
}

func (t *Directory) cert() (cert *tls.Certificate, err error) {
	if err = t.init(); err != nil {
		return nil, err
	}

	t.m.Lock()
	cert = t.cachedCert
	t.m.Unlock()

	if cert == nil {
		err = errors.Errorf("certificate missing in: %s", t.dir)
		errorsx.MaybeLog(err)
		return nil, err
	}

	return cert, nil
}

func LoadCert(pool *x509.CertPool, path string) (err error) {
	var (
		ca []byte
	)

	if envx.Boolean(false, bw.EnvLogsTLS, bw.EnvLogsVerbose) {
		log.Println("loading authority", path)
	}

	if ca, err = os.ReadFile(path); err != nil {
		return errors.Wrapf(err, "failed to read certificate: %s", path)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return nil
	}

	return nil
}

func (t *Directory) refresh() (err error) {
	var (
		certpath, keypath string
		cert              tls.Certificate
	)

	certpath = bw.LocateFirstInDir(t.dir, DefaultTLSCertServer, DefaultTLSSelfSignedCertServer)
	keypath = bw.LocateFirstInDir(t.dir, DefaultTLSKeyServer)

	if envx.Boolean(false, bw.EnvLogsTLS, bw.EnvLogsVerbose) {
		log.Println("loading certificate", certpath, keypath)
	}

	if cert, err = tls.LoadX509KeyPair(certpath, keypath); err != nil {
		return errors.WithStack(err)
	}

	t.m.Lock()
	defer t.m.Unlock()

	t.cachedCert = &cert

	if systemx.FileExists(certpath) {
		if err = LoadCert(t.pool, certpath); err != nil {
			return err
		}
	} else {
		log.Println("missing server certificate", certpath)
	}

	if systemx.FileExists(t.caFile) {
		if err = LoadCert(t.pool, t.caFile); err != nil {
			return err
		}
	} else {
		log.Println("custom certificate authority unspecified", t.caFile)
	}

	// refresh the pool
	log.Println("loading pool", t.pooldir)
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

		if err = LoadCert(t.pool, path); err != nil {
			log.Println(err)
			return nil
		} else {
			log.Println("loaded certificate", path)
		}

		return nil
	})
}
