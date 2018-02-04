package notifications

import (
	"log"
	"os"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/naoina/toml"
	"github.com/naoina/toml/ast"
)

const (
	envDeployID     = "BEARDED_WOOKIE_NOTIFICATIONS_DEPLOY_ID"
	envDeployResult = "BEARDED_WOOKIE_NOTIFICATIONS_DEPLOY_RESULT"
)

// Creator ...
type Creator func() Notifier

// Notifier ...
type Notifier interface {
	Notify(agent.DeployCommand)
}

// DecodeConfig ...
func DecodeConfig(path string, creators map[string]Creator) (n []Notifier, err error) {
	if _, err = os.Stat(path); os.IsNotExist(err) {
		n = append(n, New())
		return n, nil
	}

	tbl := decode(path)

	for name, configs := range tbl.Fields["notifications"].(*ast.Table).Fields {
		var (
			ok     bool
			plugin func() Notifier
		)

		if plugin, ok = creators[name]; !ok {
			continue
		}

		for _, config := range configs.([]*ast.Table) {
			x := plugin()
			if err = toml.UnmarshalTable(config, x); err != nil {
				log.Println("failed to load notification", name, "line:", config.Line, err)
				continue
			}
			n = append(n, x)
		}
	}

	if len(n) == 0 {
		n = append(n, New())
	}
	return n, nil
}

// ExpandEnv replaces environment variables based on the deploy command
func ExpandEnv(s string, dc agent.DeployCommand) string {
	return os.Expand(s, func(key string) string {
		switch key {
		case envDeployID:
			return bw.RandomID(dc.Archive.DeploymentID).String()
		case envDeployResult:
			return dc.Command.String()
		default:
			return os.Getenv(key)
		}
	})
}
