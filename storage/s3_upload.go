package storage

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
)

type s3P struct {
	s3Config
	*session.Session
}

func (t s3P) NewUpload(uid []byte, bytes uint64) (Uploader, error) {
	out, in := io.Pipe()
	bucket := t.s3Config.Bucket
	key := filepath.Join(t.s3Config.KeyPrefix, bw.RandomID(uid).String())

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
