package logx

import (
	"fmt"
	"log"
)

// MaybeLog logs if an error occurred.
func MaybeLog(err error) error {
	if err != nil {
		log.Output(2, fmt.Sprintln(err))
	}

	return err
}

// Verbose include stack trace
func Verbose(err error) error {
	if err != nil {
		log.Output(2, fmt.Sprintf("%+v\n", err))
	}

	return err
}
