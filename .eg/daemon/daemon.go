package daemon

import (
	"context"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/shell"
	"github.com/egdaemon/eg/runtime/x/wasi/eggolang"
)

func Install(ctx context.Context, op eg.Op) error {
	privileged := shell.Runtime().Privileged()

	return eg.Perform(
		ctx,
		eggolang.AutoInstall(),
		shell.Op(
			privileged.New("cp ~egd/go/bin/* /usr/local/bin"),
		),
	)
}

func Test(ctx context.Context, op eg.Op) error {
	return eg.Perform(
		ctx,
		eggolang.AutoCompile(),
		eg.Parallel(
			eggolang.AutoTest(),
			linting,
		),
	)
}

func linting(ctx context.Context, _ eg.Op) error {
	daemons := eggolang.Runtime().
		Environ("GOBIN", "/usr/local/bin")
	return shell.Run(
		ctx,
		daemons.New("go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest").Privileged(),
		daemons.New("golangci-lint run -v --timeout 5m"),
	)
}
