package directives

import (
	"context"
	"os"

	"github.com/james-lawrence/bw/directives/bwfs"
	"github.com/pkg/errors"
)

// ArchiveLoader directive.
type ArchiveLoader struct {
	Context
}

// Ext extensions to succeed against.
func (ArchiveLoader) Ext() []string {
	return []string{".bwfs"}
}

// Load bwfs directives from path
func (t ArchiveLoader) Load(path string) (dir Directive, err error) {
	var (
		archives []bwfs.Archive
		r        *os.File
	)

	if err = LoadsExtensions(path, "bwfs"); err != nil {
		return dir, err
	}

	if r, err = os.Open(path); err != nil {
		return dir, errors.WithStack(err)
	}
	defer r.Close()

	if archives, err = bwfs.ParseManifest(bwfs.Archive{}, r); err != nil {
		return nil, err
	}

	return closure(func(ctx context.Context) error {
		return bwfs.New(t.Context.Log, t.Context.RootDirectory).Execute(archives...)
	}), nil
}
