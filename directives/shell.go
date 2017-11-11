package directives

import (
	"io"

	"github.com/james-lawrence/bw/directives/shell"
)

// ShellLoader directive.
type ShellLoader struct {
	Context shell.Context
}

// Ext extensions to succeed against.
func (ShellLoader) Ext() []string {
	return []string{".bwcmd"}
}

// Build builds a directive from the reader.
func (t ShellLoader) Build(r io.Reader) (Directive, error) {
	var (
		err  error
		cmds []shell.Exec
	)

	if cmds, err = shell.ParseYAML(r); err != nil {
		return nil, err
	}

	return closure(func() error {
		return shell.Execute(t.Context, cmds...)
	}), nil
}
