package notifications

import (
	"fmt"
	"os"

	"github.com/naoina/toml"
	"github.com/naoina/toml/ast"
)

func decode(path string) *ast.Table {
	var (
		err   error
		raw   []byte
		table *ast.Table
	)

	if raw, err = os.ReadFile(path); err != nil {
		panic(err)
	}

	raw = []byte(deferredExpand(string(raw)))

	if table, err = toml.Parse(raw); err != nil {
		panic(err)
	}

	return table
}

func deferredExpand(s string) string {
	return os.Expand(s, func(key string) string {
		switch key {
		case EnvDeployInitiator, EnvDeployID, EnvDeployResult:
			return fmt.Sprintf("${%s}", key)
		default:
			return os.Getenv(key)
		}
	})
}
