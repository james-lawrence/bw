package directives

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

// NewWASM ...
func NewWASM() WASMLoader {
	return WASMLoader{}
}

// WASMLoader directive.
type WASMLoader struct{}

// Load wasm directive
func (t WASMLoader) Load(path string) (dir Directive, err error) {
	var (
		bin  []byte
		wasm wasmer.Instance
	)

	if err = LoadsExtensions(path, "wasm"); err != nil {
		return dir, err
	}

	return closure(func(ctx context.Context) error {
		imports := wasmer.NewImports()
		if bin, err = wasmer.ReadBytes(path); err != nil {
			return errors.Wrap(err, "failed to read wasm module")
		}

		if wasm, err = wasmer.NewInstanceWithImports(bin, imports); err != nil {
			return errors.WithStack(err)
		}
		defer wasm.Close()

		return nil
	}), nil
}
