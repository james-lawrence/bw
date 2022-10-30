package notifications

import (
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/naoina/toml"
	"github.com/naoina/toml/ast"
)

// notification environment variables.
const (
	EnvDeployID        = "BEARDED_WOOKIE_NOTIFICATIONS_DEPLOY_ID"
	EnvDeployResult    = "BEARDED_WOOKIE_NOTIFICATIONS_DEPLOY_RESULT"
	EnvDeployInitiator = "BEARDED_WOOKIE_NOTIFICATIONS_DEPLOY_INITIATOR"
	EnvDeployCommit    = "BEARDED_WOOKIE_NOTIFICATIONS_DEPLOY_COMMIT"
)

// Creator ...
type Creator func() Notifier

// Notifier ...
type Notifier interface {
	Notify(*agent.DeployCommand)
}

// DecodeConfig ...
func DecodeConfig(path string, creators map[string]Creator) (n []Notifier, err error) {
	if _, err = os.Stat(path); os.IsNotExist(err) {
		log.Println("no configuration file found, falling back to default configuration", path)
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
func ExpandEnv(s string, dc *agent.DeployCommand) string {
	return os.Expand(s, func(key string) string {
		switch key {
		case EnvDeployID:
			if dc.Archive == nil {
				log.Println("unknown archive", spew.Sdump(dc))
				return ""
			}

			return bw.RandomID(dc.Archive.DeploymentID).String()
		case EnvDeployResult:
			return dc.Command.String()
		case EnvDeployInitiator:
			return dc.GetInitiator()
		case EnvDeployCommit:
			return dc.Archive.Commit
		default:
			return os.Getenv(key)
		}
	})
}

type noop struct{}

func (t noop) Notify(dc *agent.DeployCommand) {}
