// Package protogen generates the go code for the protocol buffers in .protocol.
package protogen

import (
	"context"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/shell"
	"github.com/egdaemon/eg/runtime/x/wasi/eggolang"
)

func shellruntime() shell.Command {
	return eggolang.Runtime().Directory(egenv.WorkingDirectory())
}

// Generate the go code for the protocol buffers.
func Generate(ctx context.Context, op eg.Op) error {
	gruntime := shellruntime()
	privileged := shell.Runtime().Privileged()

	return shell.Run(
		ctx,
		// protoc is not part of the default container.
		privileged.New("command -v protoc > /dev/null || (apt-get update && apt-get install -y --no-install-recommends protobuf-compiler)"),
		// the generators are pinned by tools.go and go.mod, go install puts them in $GOPATH/bin.
		gruntime.New("go install google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc"),
		gruntime.New("protoc --version"),
		gruntime.New("protoc --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go --plugin=protoc-gen-go-grpc=$(go env GOPATH)/bin/protoc-gen-go-grpc --proto_path=.protocol --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go_out=agent --go-grpc_out=agent agent.proto"),
		gruntime.New("protoc --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go --plugin=protoc-gen-go-grpc=$(go env GOPATH)/bin/protoc-gen-go-grpc --proto_path=.protocol --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go_out=agent/acme --go-grpc_out=agent/acme acme.proto"),
		gruntime.New("protoc --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go --plugin=protoc-gen-go-grpc=$(go env GOPATH)/bin/protoc-gen-go-grpc --proto_path=.protocol --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go_out=agent/debug --go-grpc_out=agent/debug debug.proto"),
		gruntime.New("protoc --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go --plugin=protoc-gen-go-grpc=$(go env GOPATH)/bin/protoc-gen-go-grpc --proto_path=.protocol --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go_out=agent/discovery --go-grpc_out=agent/discovery discovery.proto"),
		gruntime.New("protoc --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go --proto_path=.protocol --go_opt=paths=source_relative --go_out=muxer muxer.proto"),
		gruntime.New("protoc --plugin=protoc-gen-go=$(go env GOPATH)/bin/protoc-gen-go --plugin=protoc-gen-go-grpc=$(go env GOPATH)/bin/protoc-gen-go-grpc --proto_path=.protocol --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go_out=notary --go-grpc_out=notary notary.proto"),
		gruntime.New("go fmt ./agent/... ./muxer/... ./notary/..."),
	)
}
