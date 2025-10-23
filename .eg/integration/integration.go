// Package integration runs an integration suite
package integration

import (
	"context"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/shell"
)

func Test(ctx context.Context, op eg.Op) error {
	runtime := shell.Runtime()
	return eg.Perform(
		ctx,
		shell.Op(
			runtime.New("bw me init --seed test linux-dev"),
			runtime.New("bw deploy local linux-dev"),
			runtime.New("systemctl --user restart bearded-wookie@agent1.service bearded-wookie@agent2.service bearded-wookie@agent3.service bearded-wookie@agent4.service"),
			runtime.New("env"),
			runtime.New("bw deploy env linux-test --insecure"),
		),
	)
}
