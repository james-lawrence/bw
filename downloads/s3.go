package downloads

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

// NewS3Protocol ...
func NewS3Protocol() ProtocolS3 {
	var (
		s    *session.Session
		sopt session.Options
	)

	sopt = session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:     aws.String("us-east-1"),
			HTTPClient: &http.Client{Timeout: 5 * time.Second},
		},
	}

	s = session.Must(session.NewSessionWithOptions(sopt))

	return ProtocolS3{S3: s3.New(s)}
}

// ProtocolS3 implements the registry protocol interface for s3 downloads.
type ProtocolS3 struct {
	*s3.S3
}

// Protocol ...
func (t ProtocolS3) Protocol() string {
	return s3Protocol
}

// New ...
func (t ProtocolS3) New(location string) Downloader {
	return s3d{
		S3:       t.S3,
		location: location,
	}
}

type s3d struct {
	*s3.S3
	location string
}

func (t s3d) Download() io.ReadCloser {
	var (
		err    error
		idx    int
		result *s3.GetObjectOutput
	)
	normalized := strings.TrimPrefix(t.location, s3Protocol)

	if idx = strings.IndexRune(normalized, filepath.Separator); idx == -1 {
		return newErrReader(errors.Errorf("failed to determine bucket name from: %s", t.location))
	}

	bucket, key := normalized[:idx], normalized[idx:]

	result, err = t.S3.GetObject(&s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	return maybeIO(result.Body, errors.Wrapf(err, "failed to lookup object in s3: %s, %s", bucket, key))
}
