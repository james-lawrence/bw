package directives

import (
	"github.com/james-lawrence/bw/directives/wasm"
)

// NewWASM ...
func NewWASM() WASMLoader {
	return WASMLoader{}
}

// WASMLoader directive.
type WASMLoader struct{}

// Load wasm directive
func (t WASMLoader) Load(path string) (dir Directive, err error) {
	if err = LoadsExtensions(path, "wasm"); err != nil {
		return dir, err
	}

	return wasm.Open(path)
	// return closure(func(ctx context.Context) (err error) {
	// 	var (
	// 		s  *os.File
	// 		m  *wasm.Module
	// 		vm exec.VM
	// 	)
	//
	// 	if vm, err = exec.NewVM(m); err != nil {
	// 		return errors.Wrap(err, "failed to build wasm vm")
	// 	}
	//
	// 	// imports := wasmer.NewImports()
	// 	// if bin, err = wasmer.ReadBytes(path); err != nil {
	// 	// 	return errors.Wrap(err, "failed to read wasm module")
	// 	// }
	// 	//
	// 	// if i, err = wasmer.NewInstanceWithImports(bin, imports); err != nil {
	// 	// 	return errors.WithStack(err)
	// 	// }
	// 	// defer i.Close()
	//
	// 	return nil
	// }), nil
}
