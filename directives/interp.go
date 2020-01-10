package directives

import (
	"bytes"
	"context"
	"go/build"
	"io"

	"github.com/james-lawrence/bw/directives/interp"
	"github.com/james-lawrence/bw/directives/shell"
)

// InterpLoader directive.
type InterpLoader struct {
	Context
	Environ      []string
	ShellContext shell.Context
}

// Ext extensions to match against.
func (InterpLoader) Ext() []string {
	return []string{".go"}
}

// Build builds a directive from the reader.
func (t InterpLoader) Build(r io.Reader) (Directive, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, r); err != nil {
		return nil, err
	}

	return closure(func(ctx context.Context) error {
		return interp.Compiler{
			Build:            build.Default,
			WorkingDirectory: t.Context.RootDirectory,
			Log:              t.Context.Log,
			Environ:          t.Environ,
			ShellContext:     t.ShellContext,
		}.Execute(ctx, "", buf)
	}), nil
}
