package storage

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
)

// ProtocolS3 implements the registry protocol interface for s3 downloads.
type ProtocolS3 struct {
	*s3.S3
}

// Protocol ...
func (t ProtocolS3) Protocol() string {
	return s3Protocol
}

// New ...
func (t ProtocolS3) New() Downloader {
	return s3d{
		S3: t.S3,
	}
}

type s3d struct {
	*s3.S3
}

func (t s3d) Download(ctx context.Context, a agent.Archive) io.ReadCloser {
	var (
		err    error
		idx    int
		result *s3.GetObjectOutput
	)
	normalized := strings.TrimPrefix(a.Location, s3Protocol+protocolSuffix)

	if idx = strings.IndexRune(normalized, filepath.Separator); idx == -1 {
		return newErrReader(errors.Errorf("failed to determine bucket name from: %s", a.Location))
	}

	bucket, key := normalized[:idx], normalized[idx:]

	result, err = t.S3.GetObject(&s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	return maybeIO(result.Body, errors.Wrapf(err, "failed to lookup object in s3: %s, %s", bucket, key))
}
