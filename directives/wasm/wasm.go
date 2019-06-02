package wasm

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

// Context for the WASM environment
type Context struct {
	Environ []string
	output  io.Writer
}

// Program wasm program to load an execute.
type Program struct {
	wasmer.Instance
}

func (t Program) execute(ctx context.Context, sctx Context) (err error) {
	main := t.Instance.Exports["main"]
	if _, err := main(); err != nil {
		return err
	}

	return nil
}

// Execute ...
func Execute(ctx context.Context, sctx Context, commands ...Program) error {
	for _, c := range commands {

		fmt.Fprintln(sctx.output, "executing wasm directive")
		if err := c.execute(ctx, sctx); err != nil {
			return errors.Wrapf(err, "failed to execute wasm directive")
		}

		fmt.Fprintln(sctx.output, "completed wasm directive")
	}

	return nil
}
