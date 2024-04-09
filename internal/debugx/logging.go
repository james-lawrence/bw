//go:build !debug.disabled
// +build !debug.disabled

package debugx

import (
	"fmt"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
)

// Println prints to log.Println
func Println(v ...interface{}) {
	errorsx.MaybeLog(errors.Wrap(logger.Output(2, fmt.Sprintln(v...)), "debug log failed"))
}

// Printf prints to log.Printf
func Printf(format string, v ...interface{}) {
	errorsx.MaybeLog(errors.Wrap(logger.Output(2, fmt.Sprintf(format, v...)), "debug log failed"))
}

// Print prints to log.Print
func Print(v ...interface{}) {
	errorsx.MaybeLog(errors.Wrap(logger.Output(2, fmt.Sprint(v...)), "debug log failed"))
}
