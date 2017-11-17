package storage

import (
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Config - upload configuration.
type Config struct {
	Backend string
	Options map[string]interface{}
}

// Protocol returns the protocol defined by the configuration.
func (t Config) Protocol() (_ Protocol, err error) {
	var (
		serialized []byte
	)

	if serialized, err = t.options(); err != nil {
		return nil, err
	}

	return ProtocolFromConfig(t.Backend, serialized)
}

func (t Config) options() (raw []byte, err error) {
	raw, err = yaml.Marshal(t.Options)
	return raw, errors.WithStack(err)
}
