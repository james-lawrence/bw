package blobfs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/pkg/errors"
)

const unsignedPayload = "UNSIGNED-PAYLOAD"

// NewS3 writes to the bucket under prefix using the aws default credential chain and region.
func NewS3(bucket, prefix string) (*S3, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "unable to load aws credentials")
	}

	region := "us-east-1"
	if r := sess.Config.Region; r != nil && *r != "" {
		region = *r
	}

	signer := v4.NewSigner(sess.Config.Credentials)
	signer.DisableURIPathEscaping = true // s3 requirement.

	return &S3{
		client:   http.DefaultClient,
		signer:   signer,
		endpoint: fmt.Sprintf("https://s3.%s.amazonaws.com", region),
		region:   region,
		bucket:   bucket,
		prefix:   prefix,
		ctx:      context.Background(),
	}, nil
}

// S3 is a FS backed by aws s3.
type S3 struct {
	client   *http.Client
	signer   *v4.Signer
	endpoint string
	region   string
	bucket   string
	prefix   string
	ctx      context.Context // used for Open, which has no context of its own.
}

func (t *S3) request(ctx context.Context, method, k string) (*http.Request, error) {
	u := fmt.Sprintf("%s/%s/%s", t.endpoint, url.PathEscape(t.bucket), (&url.URL{Path: k}).EscapedPath())
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Amz-Content-Sha256", unsignedPayload)

	if _, err = t.signer.Sign(req, nil, "s3", t.region, time.Now()); err != nil {
		return nil, errors.Wrap(err, "unable to sign request")
	}

	return req, nil
}

func (t *S3) Open(name string) (fs.File, error) {
	k, err := key(t.prefix, name)
	if err != nil {
		return nil, err
	}

	req, err := t.request(t.ctx, http.MethodGet, k)
	if err != nil {
		return nil, err
	}

	return openHTTP(t.client, name, req)
}

func (t *S3) OpenFile(ctx context.Context, name string) (io.WriteCloser, error) {
	k, err := key(t.prefix, name)
	if err != nil {
		return nil, err
	}

	return newCommitWriter(ctx, func(r io.Reader, size int64) error {
		req, err := t.request(ctx, http.MethodPut, k)
		if err != nil {
			return err
		}

		return put(t.client, req, r, size, fmt.Sprintf("s3://%s/%s", t.bucket, k))
	})
}
