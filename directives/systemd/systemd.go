package systemd

import (
	"context"
	"log"
	"reflect"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/envx"
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

func ensureRunning(ctx context.Context, conn *dbus.Conn, units ...string) (err error) {
	var (
		updates = make(chan *dbus.SubStateUpdate, 100)
		errs    = make(chan error, 100)
	)
	defer close(updates)

	detectFailed := func(ctx context.Context) error {
		upds, cause := conn.ListUnitsByNamesContext(ctx, units)
		if cause != nil {
			return errors.Wrap(cause, "unable to determine unit states")
		}

		for _, u := range upds {
			if envx.Boolean(false, bw.EnvLogsVerbose) {
				log.Printf("detecting unit state %+v\n", u)
			}
			switch u.ActiveState {
			case "active":
			default:
				return errors.Errorf("%s %s - %s", u.Name, u.ActiveState, u.SubState)
			}
		}

		return nil
	}
	conn.SetSubStateSubscriber(updates, errs)
	defer conn.SetSubStateSubscriber(nil, nil)

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("monitoring units initiated", units)
		defer log.Println("monitoring units completed", units)
	}

	if err = detectFailed(ctx); err != nil {
		return err
	}

	m := make(map[string]struct{}, len(units))
	for _, u := range units {
		m[u] = struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			type timeout interface {
				Timeout() bool
			}

			if x := ctx.Err(); x == context.DeadlineExceeded {
				// if the deadline passed; we still want to do the final check to ensure
				// the services are running.
				fctx, fdone := context.WithTimeout(context.Background(), 3*time.Second)
				defer fdone()
				return detectFailed(fctx)
			} else {
				return errors.WithStack(x)
			}
		case upd := <-updates:
			if _, ok := m[upd.UnitName]; !ok {
				if envx.Boolean(false, bw.EnvLogsVerbose) {
					log.Println("systemd unknown unit", upd.UnitName, m)
				}
				continue
			}

			if err = detectFailed(ctx); err != nil {
				return err
			}
		case cause := <-errs:
			return errors.WithStack(cause)
		}
	}
}

// Export functionality to the interp
func Export() (conn *dbus.Conn, exported map[string]reflect.Value, err error) {
	if conn, err = dbus.NewSystemConnection(); err != nil {
		return conn, exported, errors.Wrap(err, "failed to connect to systemd bus")
	}

	exported = map[string]reflect.Value{
		"StartUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return errors.Wrap(startJob(ctx, unit, conn.StartUnit), "systemd start unit failed")
		}),
		"RestartUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return errors.Wrap(startJob(ctx, unit, conn.RestartUnit), "systemd restart unit failed")
		}),
		"ReloadUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return errors.Wrap(startJob(ctx, unit, conn.ReloadUnit), "systemd reload unit failed")
		}),
		"RemainActive": reflect.ValueOf(func(ctx context.Context, units ...string) error {
			return errors.Wrap(ensureRunning(ctx, conn, units...), "systemd ensure service is running failed")
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
			return errors.Wrap(startJob(ctx, unit, conn.StartUnit), "systemd start unit failed")
		}),
		"RestartUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return errors.Wrap(startJob(ctx, unit, conn.RestartUnit), "systemd restart unit failed")
		}),
		"ReloadUnit": reflect.ValueOf(func(ctx context.Context, unit string) error {
			return errors.Wrap(startJob(ctx, unit, conn.ReloadUnit), "systemd reload unit failed")
		}),
		"RemainActive": reflect.ValueOf(func(ctx context.Context, units ...string) error {
			return errors.Wrap(ensureRunning(ctx, conn, units...), "systemd ensure service is running failed")
		}),
	}

	return conn, exported, nil
}
