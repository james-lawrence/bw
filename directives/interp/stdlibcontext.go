package interp

import (
	"context"
	"reflect"

	"github.com/containous/yaegi/stdlib"
)

// contextfix overrides Background() function to return the context of the deployment.
func contextfix(ctx context.Context) (exported map[string]reflect.Value) {
	exported = stdlib.Symbols["context"]
	exported["Background"] = reflect.ValueOf(func() context.Context {
		return ctx
	})

	return exported
}
