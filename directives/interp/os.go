package interp

import (
	"errors"
	"os"
	"reflect"

	"github.com/containous/yaegi/stdlib"
)

// osfix overrides the Exit function to convert it into a standard panic.
func osfix(workingdir string) (exported map[string]reflect.Value) {
	exported = stdlib.Symbols["os"]
	exported["Getwd"] = reflect.ValueOf(func() (string, error) {
		return workingdir, nil
	})
	exported["Chdir"] = reflect.ValueOf(func(path string) error {
		return &os.PathError{
			Path: path,
			Err:  errors.New("bearded-wookie disallows changing the current directory at this time"),
		}
	})
	exported["Exit"] = reflect.ValueOf(func(code int) {
		panic(1)
	})

	return exported
}
