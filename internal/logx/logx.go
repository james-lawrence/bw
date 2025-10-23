package logx

import (
	"fmt"
	"log"

	"github.com/james-lawrence/bw/internal/errorsx"
)

// Verbose include stack trace
func Verbose(err error) error {
	if err != nil {
		errorsx.Log(log.Output(2, fmt.Sprintf("%+v\n", err)))
	}

	return err
}
