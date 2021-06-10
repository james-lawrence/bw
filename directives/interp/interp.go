// Package interp provides an interpreter interface
// allowing the execution of arbitrary go code as part of the deploy.
package interp

import (
	"bytes"
	"context"
	"go/build"
	"go/format"
	"io"
	"log"
	"reflect"

	"github.com/pkg/errors"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"golang.org/x/tools/imports"

	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/directives/systemd"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/errorsx"
)

type logger interface {
	Output(depth int, s string) error
}

// Compiler used to compile and execute go code.
type Compiler struct {
	Environ          []string
	Build            build.Context
	Exports          []interp.Exports
	ShellContext     shell.Context
	WorkingDirectory string
	Log              logger
}

// Execute the script.
func (t Compiler) Execute(ctx context.Context, name string, r io.Reader) (err error) {
	var (
		formatted string
		buf       bytes.Buffer
	)

	if _, err = io.Copy(&buf, r); err != nil {
		return err
	}

	if formatted, err = _format(buf.Bytes()); err != nil {
		return err
	}

	i := interp.New(interp.Options{
		GoPath: t.Build.GOPATH,
	})
	i.Use(stdlib.Symbols)

	for _, exp := range t.Exports {
		i.Use(exp)
	}

	i.Use(interp.Exports{
		"os":                 osfix(t.WorkingDirectory), // fixes os.Exit to prevent complete destruction of the program.
		"log":                logfix(t.Log),
		"context":            contextfix(ctx),
		"bw/interp/shell":    exportShell(t.ShellContext),
		"bw/interp/env":      exportEnviron(t.Environ...),
		"bw/interp/envx":     exportEnvx(),
		"bw/interp/aws/elb":  elb(),
		"bw/interp/aws/elb2": elb2(),
	})

	if conn, exports, err := systemd.Export(); err == nil {
		defer conn.Close()
		i.Use(interp.Exports{
			"bw/interp/systemd": exports,
		})
	} else {
		log.Println("systemd disabled, unable to establish a connection")
	}

	if conn, exports, err := systemd.ExportUser(); err == nil {
		defer conn.Close()
		i.Use(interp.Exports{
			"bw/interp/systemdu": exports,
		})
	} else {
		debugx.Println("systemd disabled, unable to establish a user connection")
	}

	if err = panicSafe(func() error { return eval(ctx, i, formatted) }); err != nil {
		return errors.Wrap(err, "failed to compile")
	}

	return nil
}

func eval(ctx context.Context, i *interp.Interpreter, src string) (err error) {
	if _, err = i.Eval(src); err != nil {
		return err
	}

	return nil
}

func panicSafe(fn func() error) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		switch cause := recovered.(type) {
		case error:
			err = cause
			return
		case int:
			err = errors.Errorf("exit code: %d", cause)
			return
		case string:
			err = errors.New(cause)
			return
		default:
		}

		if res, ok := recovered.(reflect.Value); ok {
			switch v := res.Interface().(type) {
			case int:
				err = errors.Errorf("exit code: %d", v)
				return
			case string:
				err = errors.New(v)
				return
			default:
				err = errors.Errorf("recovered a panic of unknown reflected type: %s - %s - %t", res.Type(), res.Kind(), res.IsValid())
				return
			}
		}

		err = errorsx.Compact(err, errors.Errorf("recovered a panic of unknown type: %T", recovered))
	}()

	return fn()
}

// Format arbitrary source fragment.
func _format(s []byte) (_ string, err error) {
	var (
		raw []byte
	)

	if raw, err = imports.Process("generated.go", s, &imports.Options{Fragment: true, Comments: true, TabIndent: true, TabWidth: 8}); err != nil {
		return "", errors.Wrap(err, "failed to add required imports")
	}

	if raw, err = format.Source(raw); err != nil {
		return "", errors.Wrap(err, "failed to format source")
	}

	return string(raw), nil
}
