package directives

import (
	"context"
	"io"

	"github.com/james-lawrence/bw/directives/bwfs"
)

// ArchiveLoader directive.
type ArchiveLoader struct {
	Context
}

// Ext extensions to succeed against.
func (ArchiveLoader) Ext() []string {
	return []string{".bwfs"}
}

// Build builds a directive from the reader.
func (t ArchiveLoader) Build(r io.Reader) (Directive, error) {
	var (
		err      error
		archives []bwfs.Archive
	)

	if archives, err = bwfs.ParseManifest(bwfs.Archive{}, r); err != nil {
		return nil, err
	}

	return closure(func(ctx context.Context) error {
		return bwfs.New(t.Context.Log, t.Context.RootDirectory).Execute(archives...)
	}), nil
}
