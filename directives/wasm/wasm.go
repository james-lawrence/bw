package wasm

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/pkg/errors"
)

// Context for the WASM environment
type Context struct {
	Environ []string
	output  io.Writer
}

// Open a wasm directive at the given path
func Open(path string) (_ Program, err error) {
	var (
		s *os.File
		m *wasm.Module
	)

	if s, err = os.Open(path); err != nil {
		return Program{}, errors.Wrap(err, "failed to read wasm directive")
	}

	if m, err = wasm.ReadModule(s, nil); err != nil {
		return Program{}, errors.Wrap(err, "failed to read wasm module")
	}

	m.LinearMemoryIndexSpace = make([][]byte, 1024)
	return New(m), err
}

// New ...
func New(m *wasm.Module) Program {
	return Program{m: m}
}

// Program wasm program to load an execute.
type Program struct {
	m *wasm.Module
}

// Run ...
func (t Program) Run(context.Context) (err error) {
	var (
		e  *wasm.ExportEntry
		vm *exec.VM
		o  interface{}
	)

	if vm, err = exec.NewVM(t.m); err != nil {
		return errors.WithStack(err)
	}
	_ = vm
	if t.m.Export == nil {
		return errors.New("wasm module has no exported functions")
	}

	if e = t.locate(); e == nil {
		return errors.New("wasm module missing executable function")
	}

	log.Println("Executing", spew.Sdump(e))
	if fn := t.m.GetFunction(int(e.Index)); fn != nil {
		log.Println("function", spew.Sdump(fn.Sig))
	}

	if o, err = vm.ExecCode(int64(e.Index)); err != nil {
		return errors.WithStack(err)
	}

	log.Println("SUCCESS", spew.Sdump(o))

	return nil
}

func (t Program) locate() *wasm.ExportEntry {
	var (
		name string
		e    wasm.ExportEntry
	)

	for name, e = range t.m.Export.Entries {
		if name == "bw" {
			return &e
		}

		log.Println("module export", name, spew.Sdump(e))
	}

	return nil
}

//
// func (t Program) execute(ctx context.Context, sctx Context) (err error) {
// 	main := t.Instance.Exports["main"]
// 	if _, err := main(); err != nil {
// 		return err
// 	}
//
// 	return nil
// }
//
// // Execute ...
// func Execute(ctx context.Context, sctx Context, commands ...Program) error {
// 	for _, c := range commands {
// 		fmt.Fprintln(sctx.output, "executing wasm directive")
// 		if err := c.execute(ctx, sctx); err != nil {
// 			return errors.Wrapf(err, "failed to execute wasm directive")
// 		}
//
// 		fmt.Fprintln(sctx.output, "completed wasm directive")
// 	}
//
// 	return nil
// }
