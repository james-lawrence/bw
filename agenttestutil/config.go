package agenttestutil

import (
	"os"

	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// WriteAgent encodes the agent configuration into the file at path.
func WriteAgent(path string, c agent.Config) error {
	encoded, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "failed to encode configuration, try updating to a newer bearded wookie version")
	}

	return errors.Wrap(os.WriteFile(path, encoded, 0600), "failed to write configuration")
}

// WriteClient encodes the client configuration into the file at path.
func WriteClient(path string, c agent.ConfigClient) error {
	encoded, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "failed to encode configuration, try updating to a newer bearded wookie version")
	}

	return errors.Wrap(os.WriteFile(path, encoded, 0600), "failed to write configuration")
}
