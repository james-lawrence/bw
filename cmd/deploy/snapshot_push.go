package deploy

import (
	"bytes"
	"context"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/blobfs"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/pkg/errors"
)

// SnapshotPush generates a snapshot and writes the archive and its metadata to dst. objects are
// named after the bootstrap archive directory: current.tar.gz and current.meta.
// the metadata is written last so its presence implies a complete archive.
func SnapshotPush(ctx Context, dst blobfs.FS) (err error) {
	// cancelling discards anything written when the snapshot fails.
	wctx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	archive, err := dst.OpenFile(wctx, bw.BootstrapArchiveFile)
	if err != nil {
		return errors.Wrap(err, "failed to open archive")
	}

	var meta bytes.Buffer
	if err = Snapshot(ctx, iox.WriteNopCloser(archive), &meta); err != nil {
		cancel()
		archive.Close()
		return err
	}

	if err = archive.Close(); err != nil {
		return errors.Wrap(err, "failed to write archive")
	}

	md, err := dst.OpenFile(wctx, bw.BootstrapMetadataFile)
	if err != nil {
		return errors.Wrap(err, "failed to open metadata")
	}

	if _, err = md.Write(meta.Bytes()); err != nil {
		cancel()
		md.Close()
		return errors.Wrap(err, "failed to write metadata")
	}

	return errors.Wrap(md.Close(), "failed to write metadata")
}
