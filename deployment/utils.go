package deployment

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/logx"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

const deployMetadataName = "deploy.metadata"

func less(a, b agent.Deploy) bool {
	if a.Archive == nil {
		return true
	}

	if b.Archive == nil {
		return false
	}

	return a.Archive.Dts > b.Archive.Dts
}

func readAllDeployMetadata(root string) ([]agent.Deploy, error) {
	deployments := make([]agent.Deploy, 0, 10)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		var (
			d agent.Deploy
		)

		if err != nil || path == root || !info.IsDir() {
			return errors.WithStack(err)
		}

		if d, err = readDeployMetadata(filepath.Join(path, deployMetadataName)); err != nil {
			return err
		}

		deployments = append(deployments, d)

		return filepath.SkipDir
	})

	// check if the root directory does not exist.
	if os.IsNotExist(errors.Cause(err)) {
		err = nil
	}

	return deployments, err
}

func readDeployMetadata(path string) (a agent.Deploy, err error) {
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

// writeDeployMetadata writes out the archive.metadata to disk.
func writeDeployMetadata(dir string, d agent.Deploy) error {
	return writeDeployMetadataFile(filepath.Join(dir, deployMetadataName), d)
}

// writeDeployMetadata writes out the archive.metadata to disk.
func writeDeployMetadataFile(path string, d agent.Deploy) error {
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

	if raw, err = proto.Marshal(&d); err != nil {
		return errors.WithStack(err)
	}

	if _, err = io.Copy(dst, bytes.NewReader(raw)); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
