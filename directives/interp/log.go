package interp

import (
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/traefik/yaegi/stdlib"
)

// logfix overrides some basic logging, mainly changing the standard logger
func logfix(std logger) (exported map[string]reflect.Value) {
	if std == nil {
		std = log.New(io.Discard, "DISCARD", 0)
	}

	exported = stdlib.Symbols["log"]
	exported["SetOutput"] = reflect.ValueOf(func(w io.Writer) {
		errorsx.MaybeLog(std.Output(2, fmt.Sprintln("bearded-wookie: changing the default logger is not allowed at this time")))
		panic(1)
	})

	exported["SetPrefix"] = reflect.ValueOf(func(path string) error {
		errorsx.MaybeLog(std.Output(2, fmt.Sprintln("bearded-wookie: changing the default log prefix is not allowed at this time")))
		panic(1)
	})

	exported["SetFlags"] = reflect.ValueOf(func(path string) error {
		errorsx.MaybeLog(std.Output(2, fmt.Sprintln("bearded-wookie: changing the default log flags is not allowed at this time")))
		panic(1)
	})

	exported["Print"] = reflect.ValueOf(func(v ...interface{}) {
		errorsx.MaybeLog(std.Output(2, fmt.Sprint(v...)))
	})

	exported["Printf"] = reflect.ValueOf(func(format string, v ...interface{}) {
		errorsx.MaybeLog(std.Output(2, fmt.Sprintf(format, v...)))
	})

	exported["Println"] = reflect.ValueOf(func(v ...interface{}) {
		errorsx.MaybeLog(std.Output(2, fmt.Sprintln(v...)))
	})

	exported["Fatal"] = reflect.ValueOf(func(v ...interface{}) {
		msg := fmt.Sprint(v...)
		errorsx.MaybeLog(std.Output(2, msg))
		panic(msg)
	})

	exported["Fatalf"] = reflect.ValueOf(func(format string, v ...interface{}) {
		msg := fmt.Sprintf(format, v...)
		errorsx.MaybeLog(std.Output(2, msg))
		panic(msg)
	})

	exported["Fatalln"] = reflect.ValueOf(func(v ...interface{}) {
		msg := fmt.Sprintln(v...)
		errorsx.MaybeLog(std.Output(2, msg))
		panic(msg)
	})

	return exported
}
