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
	"github.com/james-lawrence/bw/internal/x/tlsx"
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
		err = errorsx.Compact(
			os.MkdirAll(t.pooldir, 0700),
			t.refresh(),
		)

		if err == nil {
			go t.background()
		}
	})

	return errors.Wrap(err, "failed to initialize certificate cache")
}

// GetCertificate for use by tls.Config.
func (t Directory) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	t.init()

	log.Println("GET SERVER CERT", hello.ServerName, hello.Conn.RemoteAddr())
	// x, _ := t.cert()
	// log.Println("SERVER CERT", tlsx.Print(x))
	return t.cert()
}

// GetClientCertificate for use by tls.Config.
func (t Directory) GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	t.init()

	// x, _ := t.cert()
	// log.Println("GET CLIENT CERT", tlsx.Print(x))
	return t.cert()
}

func (t Directory) background() {
	log.Println("WATCHING", t.dir)
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
		log.Println("$$$$$ AWAITING EVENT")
		select {
		case _ = <-t.watcher.Events:
			log.Println("watch event")
			select {
			case debounce <- struct{}{}:
			default:
			}
		case err := <-t.watcher.Errors:
			log.Println("watch error", err)
			if limit.Allow() {
				log.Println("watch error", err)
			}
		}
	}
}

func (t Directory) cert() (cert *tls.Certificate, err error) {
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

	certpath = bw.LocateFirstInDir(t.dir, DefaultTLSCertServer, DefaultTLSCertClient, DefaultTLSBootstrapCert)
	keypath = bw.LocateFirstInDir(t.dir, DefaultTLSKeyServer, DefaultTLSKeyClient)

	debugx.Println("loading", certpath, keypath)
	log.Println("##################### loading", certpath, keypath)

	if cert, err = tls.LoadX509KeyPair(certpath, keypath); err != nil {
		return errors.WithStack(err)
	}

	t.m.Lock()
	defer t.m.Unlock()
	log.Println("SERVER CERT", certpath, tlsx.Print(&cert))
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
