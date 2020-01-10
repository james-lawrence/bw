package interp

import (
	"reflect"
	"strings"
)

func exportEnviron(environ ...string) (exported map[string]reflect.Value) {
	exported = map[string]reflect.Value{}

	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		key, value := parts[0], parts[1]
		exported[key] = reflect.ValueOf((string)(value))
	}

	return exported
}
