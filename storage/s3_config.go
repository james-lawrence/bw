package storage

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

var defaultS3Config = &s3Config{
	Region:  "us-east-1",
	Timeout: 5 * time.Second,
}

type s3Config struct {
	Region    string
	Bucket    string
	KeyPrefix string `yaml:"key_prefix"`
	Timeout   time.Duration
}

func (t s3Config) Downloader() (DownloadProtocol, error) {
	var (
		s    *session.Session
		sopt session.Options
	)

	sopt = session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:     aws.String(t.Region),
			HTTPClient: &http.Client{Timeout: t.Timeout},
		},
	}

	s = session.Must(session.NewSessionWithOptions(sopt))

	return ProtocolS3{S3: s3.New(s)}, nil
}

func (t s3Config) Uploader() (_ Protocol, err error) {
	var (
		s    *session.Session
		sopt session.Options
	)

	sopt = session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:     aws.String(t.Region),
			HTTPClient: &http.Client{Timeout: t.Timeout},
		},
	}

	if s, err = session.NewSessionWithOptions(sopt); err != nil {
		return nil, errors.WithStack(err)
	}

	return s3P{
		s3Config: t,
		Session:  s,
	}, nil
}
