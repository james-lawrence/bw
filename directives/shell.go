package directives

import (
	"context"
	"os"

	"github.com/james-lawrence/bw/directives/shell"
	"github.com/pkg/errors"
)

// ShellLoader directive.
type ShellLoader struct {
	Context shell.Context
}

// Load shell directive
func (t ShellLoader) Load(path string) (dir Directive, err error) {
	var (
		cmds []shell.Exec
		r    *os.File
	)

	if err = LoadsExtensions(path, "bwcmd"); err != nil {
		return dir, err
	}

	if r, err = os.Open(path); err != nil {
		return dir, errors.WithStack(err)
	}
	defer r.Close()

	if cmds, err = shell.ParseYAML(r); err != nil {
		return nil, err
	}

	return closure(func(ctx context.Context) error {
		return shell.Execute(ctx, t.Context, cmds...)
	}), nil
}
