// Package integration runs an integration suite
package integration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/shell"
	"github.com/egdaemon/eg/runtime/x/wasi/eggolang"
	"google.golang.org/grpc/status"
)

func Test(ctx context.Context, op eg.Op) error {
	type grpcerror interface {
		error
		GRPCStatus() *status.Status
	}

	runtime := eggolang.Runtime()

	return eg.Perform(
		ctx,
		eg.Parallel(
			// shell.Op(shell.Newf("journalctl -f --user --since '-15min' | tee %s", egenv.CacheDirectory(egenv.Pattern("run-%eg.run.id%.log")))),
			eg.Sequential(
				shell.Op(
					runtime.Newf("run0 -u egd -g egd -D %s go install github.com/letsencrypt/pebble/v2/...@latest", egenv.WorkingDirectory()).Privileged(),
					runtime.Newf("run0 -u egd -g egd -D %s bw me init --seed test linux-dev", egenv.WorkingDirectory()).Privileged(),
					runtime.Newf("run0 -u egd -g egd -D %s bw deploy local linux-dev -vvv --debug", egenv.WorkingDirectory()).Privileged(),
					runtime.Newf("run0 -u egd -g egd -D %s systemctl --user enable --now bearded-wookie-pebble.service", egenv.WorkingDirectory()).Privileged().Lenient(true),
					runtime.Newf("run0 -u egd -g egd -D %s systemctl --user enable --now bearded-wookie@agent1.service bearded-wookie@agent2.service bearded-wookie@agent3.service bearded-wookie@agent4.service", egenv.WorkingDirectory()).Privileged(),
				),
				// execute various deployments to ensure behaviors.
				shell.Op(
					runtime.Newf("run0 -u egd -g egd -D %s bw deploy env linux-test-example1 --insecure", egenv.WorkingDirectory()).Privileged().Timeout(time.Minute),
				),
				// ensure deploys set exit code when they fail.
				ExpectedFailure(
					shell.Op(runtime.Newf("run0 -u egd -g egd -D %s bw deploy env linux-test-example2 --insecure", egenv.WorkingDirectory()).Privileged()),
					new(grpcerror),
				),
			),
		),
	)
}

// for commands we want to fail in a particular manner. useful for tests.
// allowed are pointers to error types (or interfaces) matched with errors.As.
func ExpectedFailure(op eg.OpFn, allowed ...any) eg.OpFn {
	return func(ctx context.Context, o eg.Op) error {
		err := op(ctx, o)
		if err == nil {
			return errors.New("expected an error but operation succeeded")
		}

		for _, target := range allowed {
			if errors.As(err, target) {
				return nil
			}
		}

		return fmt.Errorf("an unexpected error occurred: %w", err)
	}
}
