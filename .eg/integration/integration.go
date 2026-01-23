// Package integration runs an integration suite
package integration

import (
	"context"
	"fmt"

	"eg/compute/errorsx"

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
		shell.Op(
			runtime.Newf("run0 -u egd -g egd -D %s go install github.com/letsencrypt/pebble/v2/...@latest", egenv.WorkingDirectory()).Privileged(),
			runtime.Newf("run0 -u egd -g egd -D %s bw me init --seed test linux-dev", egenv.WorkingDirectory()).Privileged(),
			runtime.Newf("run0 -u egd -g egd -D %s bw deploy local linux-dev", egenv.WorkingDirectory()).Privileged(),
			runtime.Newf("run0 -u egd -g egd -D %s systemctl --user enable --now bearded-wookie-pebble.service", egenv.WorkingDirectory()).Privileged().Lenient(true),
			runtime.Newf("run0 -u egd -g egd -D %s systemctl --user enable --now bearded-wookie@agent1.service bearded-wookie@agent2.service bearded-wookie@agent3.service bearded-wookie@agent4.service", egenv.WorkingDirectory()).Privileged(),
		),
		// execute various deployments to ensure behaviors.
		shell.Op(runtime.Newf("run0 -u egd -g egd -D %s bw deploy env linux-test-example1 --insecure", egenv.WorkingDirectory()).Privileged()),
		// ensure deploys set exit code when they fail.
		ExpectedFailure(
			shell.Op(runtime.Newf("run0 -u egd -g egd -D %s bw deploy env linux-test-example2 --insecure", egenv.WorkingDirectory()).Privileged()),
			new(grpcerror),
		),
	)
}

// for commands we want to fail in a particular manner. useful for tests.
func ExpectedFailure(op eg.OpFn, allowed ...any) eg.OpFn {
	return func(ctx context.Context, o eg.Op) error {
		if err := op(ctx, o); err == nil {
			return fmt.Errorf("unexpected success")
		} else if errorsx.Ignore2(err, allowed...) != nil {
			return err
		}
		return nil
	}
}
