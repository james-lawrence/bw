// Package integration runs an integration suite
package integration

import (
	"context"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/shell"
	"github.com/egdaemon/eg/runtime/x/wasi/eggolang"
)

func Test(ctx context.Context, op eg.Op) error {
	runtime := shell.Runtime()

	return eg.Perform(
		ctx,
		shell.Op(
			eggolang.Runtime().Newf("run0 -u egd -g egd -D %s go install github.com/letsencrypt/pebble/v2/...@latest", egenv.WorkingDirectory()).Privileged(),
			runtime.Newf("run0 -u egd -g egd -D %s bw me init --seed test linux-dev", egenv.WorkingDirectory()).Privileged(),
			runtime.Newf("run0 -u egd -g egd -D %s bw deploy local linux-dev", egenv.WorkingDirectory()).Privileged(),
			runtime.Newf("run0 -u egd -g egd -D %s systemctl --user enable --now bearded-wookie-pebble.service", egenv.WorkingDirectory()).Privileged().Lenient(true),
			runtime.Newf("run0 -u egd -g egd -D %s systemctl --user enable --now bearded-wookie@agent1.service bearded-wookie@agent2.service bearded-wookie@agent3.service bearded-wookie@agent4.service", egenv.WorkingDirectory()).Privileged(),
			runtime.Newf("run0 -u egd -g egd -D %s bw deploy env linux-test --insecure", egenv.WorkingDirectory()).Privileged(),
		),
	)
}
