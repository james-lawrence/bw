package interp

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/containous/yaegi/stdlib"
)

// logfix overrides some basic logging, mainly changing the standard logger
func logfix(std logger) (exported map[string]reflect.Value) {
	if std == nil {
		std = log.New(ioutil.Discard, "DISCARD", 0)
	}

	exported = stdlib.Symbols["log"]
	exported["SetOutput"] = reflect.ValueOf(func(w io.Writer) {
		std.Output(2, fmt.Sprintln("bearded-wookie: changing the default logger is not allowed at this time"))
		panic(1)
	})

	exported["SetPrefix"] = reflect.ValueOf(func(path string) error {
		std.Output(2, fmt.Sprintln("bearded-wookie: changing the default log prefix is not allowed at this time"))
		panic(1)
	})

	exported["SetFlags"] = reflect.ValueOf(func(path string) error {
		std.Output(2, fmt.Sprintln("bearded-wookie: changing the default log flags is not allowed at this time"))
		panic(1)
	})

	exported["Print"] = reflect.ValueOf(func(v ...interface{}) {
		std.Output(2, fmt.Sprint(v...))
	})

	exported["Printf"] = reflect.ValueOf(func(format string, v ...interface{}) {
		std.Output(2, fmt.Sprintf(format, v...))
	})

	exported["Println"] = reflect.ValueOf(func(v ...interface{}) {
		std.Output(2, fmt.Sprintln(v...))
	})

	exported["Fatal"] = reflect.ValueOf(func(v ...interface{}) {
		std.Output(2, fmt.Sprint(v...))
		panic(1)
	})

	exported["Fatalf"] = reflect.ValueOf(func(format string, v ...interface{}) {
		std.Output(2, fmt.Sprintf(format, v...))
		panic(1)
	})

	exported["Fatalln"] = reflect.ValueOf(func(v ...interface{}) {
		std.Output(2, fmt.Sprintln(v...))
		panic(1)
	})

	return exported
}