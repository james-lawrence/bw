// Package integration runs an integration suite
package integration

import (
	"context"
	"time"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/shell"
	"github.com/egdaemon/eg/runtime/x/wasi/egbug"
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
			shell.Op(shell.Newf("journalctl -f --user --since '-15min' | tee %s", egenv.CacheDirectory(egenv.RunID()+".log"))),
			egbug.DebugFailure(eg.Sequential(
				shell.Op(
					runtime.Newf("run0 -u egd -g egd -D %s go install github.com/letsencrypt/pebble/v2/...@latest", egenv.WorkingDirectory()).Privileged(),
					runtime.Newf("run0 -u egd -g egd -D %s bw me init --seed test linux-dev", egenv.WorkingDirectory()).Privileged(),
					runtime.Newf("run0 -u egd -g egd -D %s bw deploy local linux-dev", egenv.WorkingDirectory()).Privileged(),
					runtime.Newf("run0 -u egd -g egd -D %s systemctl --user enable --now bearded-wookie-pebble.service", egenv.WorkingDirectory()).Privileged().Lenient(true),
					runtime.Newf("run0 -u egd -g egd -D %s systemctl --user enable --now bearded-wookie@agent1.service bearded-wookie@agent2.service bearded-wookie@agent3.service bearded-wookie@agent4.service", egenv.WorkingDirectory()).Privileged(),
				),
				// execute various deployments to ensure behaviors.
				shell.Op(
					runtime.Newf("run0 -u egd -g egd -D %s bw deploy env linux-test-example1 --insecure", egenv.WorkingDirectory()).Privileged().Timeout(time.Minute),
				),
				// // ensure deploys set exit code when they fail.
				// ExpectedFailure(
				// 	shell.Op(runtime.Newf("run0 -u egd -g egd -D %s bw deploy env linux-test-example2 --insecure", egenv.WorkingDirectory()).Privileged()),
				// 	new(grpcerror),
				// ),
			), diagnostics),
		),
	)
}

// diagnostics collects what the services have to say, the snapshot taken when the test starts
// predates them. failures are lenient, this runs when something has already gone wrong.
func diagnostics(ctx context.Context, _ eg.Op) error {
	return shell.Run(
		ctx,
		shell.New("systemctl --user status 'bearded-wookie@agent*' bearded-wookie-pebble --no-pager").Lenient(true),
		shell.New("systemctl --user show 'bearded-wookie@agent*' -p Id -p ActiveState -p Result -p ExecMainStatus -p NRestarts --no-pager").Lenient(true),
		// agents that can't join retry in a tight loop and flood the journal with bootstrap lines, drop them or the failure is buried.
		shell.New(`journalctl --user --no-pager --since '-15min' -u 'bearded-wookie@agent*' -u bearded-wookie-pebble | grep -vE 'bootstrap\.go:|\]: +(\*|\(string\)|\})' | tail -400`).Lenient(true),
	)
}
