package directives

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}

// Context global context for the agent.
type Context struct {
	Log           logger
	RootDirectory string
}

// Directive ...
type Directive interface {
	Run() error
}

// Load the directives from the provided directory.
func Load(l logger, dir string, loaders ...Loader) ([]Directive, error) {
	var (
		err error
	)

	extmap := loaderToExts(l, loaders...)
	results := make([]Directive, 0, 64)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		var (
			found     bool
			loader    Loader
			directive Directive
			reader    *os.File
		)

		if err != nil {
			return err
		}

		// don't try to process a directory as a directive, instead
		// recurse into the directory.
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if loader, found = extmap[ext]; !found {
			l.Println("no directive exists for", ext, ":", path, "skipping")
			return nil
		}

		if reader, err = os.Open(path); err != nil {
			return errors.Wrapf(err, "failed to open: %s", path)
		}
		defer reader.Close()

		if directive, err = loader.Build(reader); err != nil {
			return errors.Wrapf(err, "failed to build directive for: %s", path)
		}

		results = append(results, directive)
		return nil
	})

	return results, err
}

func loaderToExts(logger logger, loaders ...Loader) map[string]Loader {
	m := make(map[string]Loader, len(loaders))
	for _, l := range loaders {
		for _, ext := range l.Ext() {
			if _, found := m[ext]; found {
				logger.Println("extension is already mapped ignoring")
			} else {
				m[ext] = l
			}
		}
	}

	return m
}

// Loader represents a directive to be used for the specified extensions.
type Loader interface {
	// extensions to match against.
	Ext() []string
	Build(io.Reader) (Directive, error)
}

type closure func() error

func (t closure) Run() error {
	return t()
}

// NoopLoader ...
type NoopLoader struct{}

// Build ...
func (t NoopLoader) Build(r io.Reader) (Directive, error) {
	return NoopDirective{}, nil
}

// Ext ...
func (t NoopLoader) Ext() []string {
	return []string(nil)
}

// NoopDirective ...
type NoopDirective struct{}

// Run - implements Directive interface.
func (t NoopDirective) Run() error {
	return nil
}
