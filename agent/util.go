package agent

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/james-lawrence/bw/internal/x/logx"
)

// ReadMetadata from the specified file.
func ReadMetadata(path string) (a *DeployCommand, err error) {
	var (
		raw []byte
	)

	a = &DeployCommand{}

	if raw, err = ioutil.ReadFile(path); err != nil {
		return a, errors.WithStack(err)
	}

	if err = proto.Unmarshal(raw, a); err != nil {
		return a, errors.WithStack(err)
	}

	return a, nil
}

// WriteMetadata to the specified file
func WriteMetadata(path string, d *DeployCommand) error {
	var (
		err error
		dst *os.File
		raw []byte
	)

	if dst, err = os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err != nil {
		return errors.WithStack(err)
	}
	defer func() { logx.MaybeLog(errors.WithMessage(dst.Close(), "failed to close archive metadata file")) }()
	defer func() { logx.MaybeLog(errors.WithMessage(dst.Sync(), "failed to sync archive metadata to disk")) }()

	if raw, err = proto.Marshal(d); err != nil {
		return errors.WithStack(err)
	}

	if _, err = io.Copy(dst, bytes.NewReader(raw)); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
