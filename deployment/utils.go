package deployment

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// writeArchiveMetadata writes out the archive.metadata to disk.
func writeArchiveMetadata(dctx DeployContext) error {
	var (
		err error
		dst *os.File
		raw []byte
	)

	if dst, err = os.OpenFile(filepath.Join(dctx.Root, "archive.metadata"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err != nil {
		return errors.WithStack(err)
	}
	defer func() { logErr(errors.WithMessage(dst.Sync(), "failed to sync archive metadata to disk")) }()
	defer func() { logErr(errors.WithMessage(dst.Close(), "failed to close archive metadata file")) }()

	if raw, err = proto.Marshal(&dctx.Archive); err != nil {
		return errors.WithStack(err)
	}

	if _, err = io.Copy(dst, bytes.NewReader(raw)); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
