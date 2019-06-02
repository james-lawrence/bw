package directives

import (
	"context"

	"github.com/james-lawrence/bw/directives/shell"
	"github.com/pkg/errors"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

// WASMLoader directive.
type WASMLoader struct {
	Context shell.Context
}

// Load wasm directive
func (t WASMLoader) Load(path string) (dir Directive, err error) {
	var (
		bin  []byte
		wasm wasmer.Instance
	)

	if err = LoadsExtensions(path, "wasm"); err != nil {
		return dir, err
	}

	imports := wasmer.NewImports()
	if bin, err = wasmer.ReadBytes(path); err != nil {
		return dir, errors.WithStack(err)
	}
	if wasm, err = wasmer.NewInstanceWithImports(bin, imports); err != nil {
		return dir, errors.WithStack(err)
	}
	defer wasm.Close()
	return closure(func(ctx context.Context) error {
		return nil
	}), nil
}
