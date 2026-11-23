package blobfs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
)

const gcsScopeReadWrite = "https://www.googleapis.com/auth/devstorage.read_write"

// NewGCS writes to the bucket under prefix using the google application default credentials.
func NewGCS(ctx context.Context, bucket, prefix string) (*GCS, error) {
	c, err := google.DefaultClient(ctx, gcsScopeReadWrite)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load google credentials")
	}

	return &GCS{
		client:   c,
		endpoint: "https://storage.googleapis.com",
		bucket:   bucket,
		prefix:   prefix,
		ctx:      ctx,
	}, nil
}

// GCS is a FS backed by google cloud storage.
type GCS struct {
	client   *http.Client
	endpoint string
	bucket   string
	prefix   string
	ctx      context.Context // used for Open, which has no context of its own.
}

func (t *GCS) Open(name string) (fs.File, error) {
	k, err := key(t.prefix, name)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(t.ctx, http.MethodGet, fmt.Sprintf("%s/storage/v1/b/%s/o/%s?alt=media", t.endpoint, url.PathEscape(t.bucket), url.PathEscape(k)), nil)
	if err != nil {
		return nil, err
	}

	return openHTTP(t.client, name, req)
}

func (t *GCS) OpenFile(ctx context.Context, name string) (io.WriteCloser, error) {
	k, err := key(t.prefix, name)
	if err != nil {
		return nil, err
	}

	return newCommitWriter(ctx, func(r io.Reader, size int64) error {
		q := url.Values{"uploadType": {"media"}, "name": {k}}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/upload/storage/v1/b/%s/o?%s", t.endpoint, url.PathEscape(t.bucket), q.Encode()), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		return put(t.client, req, r, size, fmt.Sprintf("gs://%s/%s", t.bucket, k))
	})
}
