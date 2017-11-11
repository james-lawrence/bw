package packagekit

import (
	"fmt"
	"time"
)

// Error packagekit error.
type Error interface {
	error
	Code() ErrorEnum
}

func newError(code ErrorEnum, msg string) error {
	return transactionError{
		code: code,
		msg:  msg,
	}
}

type transactionError struct {
	msg  string
	code ErrorEnum
}

func (t transactionError) Code() ErrorEnum {
	return t.code
}

func (t transactionError) Error() string {
	return fmt.Sprintf("%s(%d): %s", t.code, t.code, t.msg)
}

type exitError struct {
	code     ExitEnum
	duration time.Duration
}

func (t exitError) Error() string {
	return fmt.Sprintf("%s(%d): %s", t.code, t.code, t.duration)
}
