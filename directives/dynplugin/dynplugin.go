package dynplugin

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"plugin"

	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/directives"
)

const (
	pluginBuilderFunction = "NewDirective"
)

// Directive ...
type Directive interface {
	// extensions to match against.
	Ext() []string
	Build(io.Reader) (directives.Directive, error)
}

// Load directives from directory.
func Load(dir string) (results []Directive, err error) {
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		var (
			ok        bool
			plug      *plugin.Plugin
			sym       plugin.Symbol
			directive func() Directive
		)

		if err != nil {
			return err
		}

		// ignore subdirectories.
		if info.IsDir() {
			if path != dir {
				log.Println("skipping directory", path)
				return filepath.SkipDir
			}

			return nil
		}

		if plug, err = plugin.Open(path); err != nil {
			return errors.WithStack(err)
		}

		if sym, err = plug.Lookup(pluginBuilderFunction); err != nil {
			return errors.WithStack(err)
		}

		if directive, ok = sym.(func() Directive); !ok {
			return errors.New("invalid plugin")
		}

		results = append(results, directive())
		return nil
	})

	return results, err
}
