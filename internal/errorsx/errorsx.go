package errorsx

import (
	"errors"
	"fmt"
	"log"
	"time"
)

// Wrap annotates err with the message, the result can be unwrapped to the original error.
// returns nil if err is nil.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf annotates err with the formatted message, the result can be unwrapped to the original error.
// returns nil if err is nil.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// Is reports whether any error in cause's tree matches any of the protos, according to errors.Is.
func Is(cause error, protos ...error) bool {
	for _, proto := range protos {
		if errors.Is(cause, proto) {
			return true
		}
	}

	return false
}

// Compact returns the first error in the set, if any.
func Compact(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

// Zero logs that the error occurred but otherwise ignores it, returning the value.
func Zero[T any](v T, err error) T {
	Log(err)
	return v
}

// From discards the value and returns the error, for consuming (value, error) results
// when only the error matters, e.g.) w.CloseWithError(errorsx.From(io.Copy(w, r))).
func From[T any](_ T, err error) error {
	return err
}

func Log(err error) {
	if err == nil {
		return
	}

	if cause := log.Output(2, fmt.Sprintln(err)); cause != nil {
		panic(cause)
	}
}

func Fatal(err error) {
	if err == nil {
		return
	}

	panic(err)
}

func Ignore(err error, ignore ...error) error {
	for _, i := range ignore {
		if errors.Is(err, i) {
			return nil
		}
	}

	return err
}

// CompactMonad an error that collects and returns the first error encountered.
type CompactMonad struct {
	cause error
}

func (t CompactMonad) Error() string { return t.cause.Error() }

// Cause implement errors.Cause interface.
func (t CompactMonad) Cause() error { return t.cause }

// Compact returns a monad holding the first error
// encountered.
func (t CompactMonad) Compact(errs ...error) CompactMonad {
	if t.cause != nil {
		return t
	}

	return CompactMonad{cause: Compact(errs...)}
}

// String useful wrapper for string constants as errors.
type String string

func (t String) Error() string {
	return string(t)
}

// Timeout error.
type Timeout interface {
	error
	Timedout() time.Duration
}

// Timedout represents a timeout.
func Timedout(cause error, d time.Duration) error {
	if cause == nil {
		return nil
	}

	return timeout{
		error: cause,
		d:     d,
	}
}

type timeout struct {
	error
	d time.Duration
}

func (t timeout) Timedout() time.Duration {
	return t.d
}

// Notification presents an error that will be displayed to the user
// to provide notifications.
func Notification(err error) error {
	return notification{
		error: err,
	}
}

type notification struct {
	error
}

func (t notification) Notification() {}
func (t notification) Unwrap() error {
	return t.error
}
func (t notification) Cause() error {
	return t.error
}

// UserFriendly represents an error that will be displayed to users.
func UserFriendly(err error) error {
	return userfriendly{
		error: err,
	}
}

type userfriendly struct {
	error
}

// user friendly error
func (t userfriendly) UserFriendly() {}
func (t userfriendly) Unwrap() error {
	return t.error
}
func (t userfriendly) Cause() error {
	return t.error
}
