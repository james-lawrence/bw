package uploads

import (
	yaml "gopkg.in/yaml.v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
		svc:      s3.New(s, aws.NewConfig()),
	}, nil
}

type s3config struct {
	Bucket    string
	KeyPrefix string `yaml:"key_prefix"`
}

type s3P struct {
	s3config
	svc *s3.S3
}

func (t s3P) NewUpload(uid []byte, bytes uint64) (Uploader, error) {
	return nil, errors.New("not implemented")
}
