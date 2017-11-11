package deployment

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw/agent"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

const archiveMetadataName = "archive.metadata"

func archivePointers(archives ...agent.Archive) []*agent.Archive {
	out := make([]*agent.Archive, 0, len(archives))
	for _, a := range archives {
		tmp := a
		out = append(out, &tmp)
	}
	return out
}

func readAllArchiveMetadata(root string) ([]agent.Archive, error) {
	archives := make([]agent.Archive, 0, 5)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		var (
			a agent.Archive
		)

		if err != nil || path == root || !info.IsDir() {
			return errors.WithStack(err)
		}

		if a, err = readArchiveMetadata(filepath.Join(path, archiveMetadataName)); err != nil {
			return err
		}

		archives = append(archives, a)

		return filepath.SkipDir
	})

	return archives, err
}

func readArchiveMetadata(path string) (a agent.Archive, err error) {
	var (
		raw []byte
	)

	if raw, err = ioutil.ReadFile(path); err != nil {
		return a, errors.WithStack(err)
	}

	if err = proto.Unmarshal(raw, &a); err != nil {
		return a, errors.WithStack(err)
	}

	return a, nil
}

// writeArchiveMetadata writes out the archive.metadata to disk.
func writeArchiveMetadata(dctx DeployContext) error {
	var (
		err error
		dst *os.File
		raw []byte
	)

	if dst, err = os.OpenFile(filepath.Join(dctx.Root, archiveMetadataName), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err != nil {
		return errors.WithStack(err)
	}
	defer func() { logErr(errors.WithMessage(dst.Close(), "failed to close archive metadata file")) }()
	defer func() { logErr(errors.WithMessage(dst.Sync(), "failed to sync archive metadata to disk")) }()

	if raw, err = proto.Marshal(&dctx.Archive); err != nil {
		return errors.WithStack(err)
	}

	if _, err = io.Copy(dst, bytes.NewReader(raw)); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
