package packagekit

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type dbusError struct {
	namespace string
	desc      string
}

func (t dbusError) Error() string {
	return fmt.Sprintf("%s: %s", t.namespace, t.desc)
}

// IgnoreNotSupported returns nil if the error is ErrorNotSupported
func IgnoreNotSupported(err error) error {
	switch cause := errors.Cause(err).(type) {
	case dbusError:
		switch cause.namespace {
		case "org.freedesktop.PackageKit.Transaction.NotSupported":
			return nil
		default:
			return err
		}
	default:
		return err
	}
}

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
