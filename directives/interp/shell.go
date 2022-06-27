package interp

import (
	"context"
	"reflect"

	"github.com/james-lawrence/bw/directives/shell"
)

func exportShell(sctx shell.Context) (exported map[string]reflect.Value) {
	exported = map[string]reflect.Value{
		"Lenient":    reflect.ValueOf(shell.OptionLenient),
		"Environ":    reflect.ValueOf(shell.OptionAppendEnviron),
		"Timeout":    reflect.ValueOf(shell.OptionTimeout),
		"WorkingDir": reflect.ValueOf(shell.OptionDir),
		"TempDir":    reflect.ValueOf(shell.OptionTempDir),
		"CacheDir":   reflect.ValueOf(shell.OptionCacheDir),
		"LoadEnv":    reflect.ValueOf(shell.OptionLoadEnv),
		"Run": reflect.ValueOf(func(ctx context.Context, cmd string, options ...shell.Option) error {
			return shell.Execute(ctx, shell.NewContext(sctx, options...), shell.Exec{Command: cmd})
		}),
	}

	return exported
}
