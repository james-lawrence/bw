package uploads

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v1"

	"github.com/james-lawrence/bw"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

func newS3PFromConfig(serialized []byte) (_ Protocol, err error) {
	var (
		cfg  s3config
		s    *session.Session
		sopt session.Options
	)

	if err = errors.WithStack(yaml.Unmarshal(serialized, &cfg)); err != nil {
		return nil, err
	}

	sopt = session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:     aws.String(cfg.Region),
			HTTPClient: &http.Client{Timeout: 5 * time.Second},
		},
	}

	if s, err = session.NewSessionWithOptions(sopt); err != nil {
		return nil, errors.WithStack(err)
	}

	return s3P{
		s3config: cfg,
		Session:  s,
	}, nil
}

type s3config struct {
	Bucket    string
	KeyPrefix string `yaml:"key_prefix"`
	Region    string
}

type s3P struct {
	s3config
	*session.Session
}

func (t s3P) NewUpload(uid []byte, bytes uint64) (Uploader, error) {
	out, in := io.Pipe()
	bucket := t.s3config.Bucket
	key := filepath.Join(t.s3config.KeyPrefix, bw.RandomID(uid).String())

	return s3u{
		dst: in,
		sha: sha256.New(),
		o:   &sync.Once{},
		s3u: s3manager.NewUploader(t.Session),
		s3ui: s3manager.UploadInput{
			Bucket: &bucket,
			Key:    &key,
			Body:   out,
		},
		bytes:   bytes,
		failure: make(chan error),
		upload:  make(chan *s3manager.UploadOutput),
	}, nil
}

type s3u struct {
	bytes   uint64
	sha     hash.Hash
	dst     io.WriteCloser
	o       *sync.Once
	s3ui    s3manager.UploadInput
	s3u     *s3manager.Uploader
	failure chan error
	upload  chan *s3manager.UploadOutput
}

func (t s3u) Upload(r io.Reader) (hash.Hash, error) {
	t.o.Do(func() {
		go t.background()
	})
	return upload(r, t.sha, t.dst)
}

// Info ...
func (t s3u) Info() (hash.Hash, string, error) {
	defer func() {
		close(t.failure)
		close(t.upload)
	}()
	t.dst.Close()

	select {
	case err := <-t.failure:
		return nil, "", errors.Wrap(err, "failed upload archive")
	case _ = <-t.upload:
		return t.sha, fmt.Sprintf("s3://%s", filepath.Join(*t.s3ui.Bucket, *t.s3ui.Key)), nil
	}
}

func (t s3u) background() {
	var (
		err    error
		upload *s3manager.UploadOutput
	)
	opt := func(u *s3manager.Uploader) {
		// TODO: verify this calculation its probably rounding wrong.
		u.PartSize = int64(t.bytes / s3manager.MaxUploadParts)
	}

	if upload, err = t.s3u.Upload(&t.s3ui, opt); err != nil {
		t.failure <- errors.WithStack(err)
		return
	}

	t.upload <- upload
}
