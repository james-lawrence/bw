package peering

import (
	"context"
	"os"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

// File based peering
type File struct {
	Path string
}

// Peers - reads peers from a file.
func (t File) Peers(context.Context) (results []string, err error) {
	var (
		data []byte
	)

	if _, err = os.Stat(t.Path); os.IsNotExist(err) {
		return results, nil
	}

	if data, err = os.ReadFile(t.Path); err != nil {
		return results, errors.Wrapf(err, "failed to peers from file: %s", t.Path)
	}

	err = errors.Wrap(yaml.Unmarshal(data, &results), "failed to load peers from file")
	return results, err
}

// Snapshot - writes peers to a file.
func (t File) Snapshot(peers []string) error {
	var (
		err  error
		data []byte
	)

	if data, err = yaml.Marshal(peers); err != nil {
		return errors.Wrap(err, "failed to marshal peers")
	}

	return errors.Wrapf(os.WriteFile(t.Path, data, 0600), "failed to write file: %s", t.Path)
}
