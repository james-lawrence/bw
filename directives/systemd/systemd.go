package systemd

import (
	"context"
	"reflect"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/pkg/errors"
)

func resultToError(result string) error {
	if result == "done" {
		return nil
	}
	return errors.New(result)
}

func startJob(ctx context.Context, target string, d func(string, string, chan<- string) (int, error)) error {
	await := make(chan string)

	_, err := d(target, "replace", await)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-await:
		return resultToError(result)
	}
}

// Export functionality to the interp
func Export() (conn *dbus.Conn, exported map[string]reflect.Value, err error) {
	if conn, err = dbus.NewSystemConnection(); err != nil {
		return conn, exported, errors.Wrap(err, "failed to connect to systemd bus")
	}

	exported = map[string]reflect.Value{
		"StartUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return startJob(ctx, unit, conn.StartUnit)
		}),
		"RestartUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return startJob(ctx, unit, conn.RestartUnit)
		}),
		"ReloadUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return startJob(ctx, unit, conn.ReloadUnit)
		}),
	}

	return conn, exported, nil
}

// ExportUser functionality to the interp
func ExportUser() (conn *dbus.Conn, exported map[string]reflect.Value, err error) {
	if conn, err = dbus.NewUserConnection(); err != nil {
		return conn, exported, errors.Wrap(err, "failed to connect to systemd bus")
	}

	exported = map[string]reflect.Value{
		"StartUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return startJob(ctx, unit, conn.StartUnit)
		}),
		"RestartUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return startJob(ctx, unit, conn.RestartUnit)
		}),
		"ReloadUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return startJob(ctx, unit, conn.ReloadUnit)
		}),
	}

	return conn, exported, nil
}
