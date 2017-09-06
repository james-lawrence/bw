package uploads

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"path/filepath"

	yaml "gopkg.in/yaml.v1"

	"bitbucket.org/jatone/bearded-wookie"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

func newS3PFromConfig(serialized []byte) (_ Protocol, err error) {
	var (
		cfg s3config
		s   *session.Session
	)

	if err = errors.WithStack(yaml.Unmarshal(serialized, &cfg)); err != nil {
		return nil, err
	}

	if s, err = session.NewSession(); err != nil {
		return nil, errors.WithStack(err)
	}

	return s3P{
		s3config: cfg,
		Uploader: s3manager.NewUploader(s),
	}, nil
}

type s3config struct {
	Bucket    string
	KeyPrefix string `yaml:"key_prefix"`
}

type s3P struct {
	s3config
	*s3manager.Uploader
}

func (t s3P) NewUpload(uid []byte, bytes uint64) (Uploader, error) {
	out, in := io.Pipe()
	bucket := t.s3config.Bucket
	key := filepath.Join(t.s3config.KeyPrefix, bw.RandomID(uid).String())
	upload, err := t.Uploader.Upload(&s3manager.UploadInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   out,
	}, func(u *s3manager.Uploader) {
		// TODO: verify this calculation its probably rounding wrong.
		u.PartSize = int64(bytes / s3manager.MaxUploadParts)
	})

	if err != nil {
		out.Close()
		in.Close()
		return nil, errors.WithStack(err)
	}

	return s3u{
		upload: upload,
		dst:    in,
		sha:    sha256.New(),
	}, nil
}

type s3u struct {
	sha    hash.Hash
	dst    io.WriteCloser
	upload *s3manager.UploadOutput
}

func (t s3u) Upload(r io.Reader) (hash.Hash, error) {
	return upload(r, t.sha, t.dst)
}

// Info ...
func (t s3u) Info() (hash.Hash, string, error) {
	if err := t.dst.Close(); err != nil {
		return nil, "", errors.Wrap(err, "failed to close upload")
	}

	return t.sha, fmt.Sprintf("s3://%s", t.upload.Location), nil
}
