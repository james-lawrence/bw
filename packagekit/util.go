package packagekit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus"
	"github.com/pkg/errors"
)

func mapPackageID(pset ...Package) (ids []string) {
	ids = make([]string, 0, len(pset))
	for _, p := range pset {
		ids = append(ids, p.ID)
	}
	return ids
}

func awaitCompletion(ctx context.Context, c chan *dbus.Signal) (time.Duration, error) {
	start := time.Now()
	for {
		select {
		case _ = <-ctx.Done():
			return time.Now().Sub(start), errors.Wrap(ctx.Err(), "operation timed out")
		case event := <-c:
			switch event.Name {
			case signalTransactionFinished:
				err := handleFinished(event)
				duration := exitDuration(err)
				return duration, ignoreSuccess(err)
			case signalTransactionError:
				return time.Now().Sub(start), signalError(event)
			case signalTransactionDestroy:
				return time.Now().Sub(start), nil
			}
		}
	}
}

func awaitEvent(ctx context.Context, c chan *dbus.Signal) (*dbus.Signal, error) {
	select {
	case _ = <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "operation timed out")
	case event := <-c:
		switch event.Name {
		case signalTransactionFinished:
			return event, ignoreSuccess(handleFinished(event))
		case signalTransactionError:
			return event, signalError(event)
		}
		return event, nil
	}
}

func propertiesSignal(rules ...string) string {
	return fmt.Sprintf("type='signal',interface='org.freedesktop.DBus.Properties',%s", strings.Join(rules, ","))
}

func transactionSignal(rules ...string) string {
	signal := fmt.Sprintf("type='signal',interface='%s',%s", pkTransactionDbusInterface, strings.Join(rules, ","))
	return signal
}

func signalError(signal *dbus.Signal) error {
	var (
		err  string
		code uint32
	)

	if decodeErr := errors.Wrapf(dbus.Store(signal.Body, &code, &err), "failed to decode %s", signalTransactionError); decodeErr != nil {
		return decodeErr
	}

	return transactionError{code: ErrorEnum(code), msg: err}
}

func parseDBusError(err error) error {
	switch cause := err.(type) {
	case dbus.Error:
		var (
			desc string
		)

		// ignoring
		if err = dbus.Store(cause.Body, &desc); err != nil {
			fmt.Printf("decode error: %T: %v\n", err, err)
			return errors.WithStack(err)
		}
		return dbusError{namespace: cause.Name, desc: desc}
	default:
		fmt.Printf("unknown error: %T: %v\n", err, err)
		return errors.WithStack(err)
	}
}
