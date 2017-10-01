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
func Load(l logger, dir string, loaders ...loader) ([]Directive, error) {
	var (
		err error
	)

	extmap := loaderToExts(l, loaders...)
	results := make([]Directive, 0, 1024)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		var (
			found     bool
			_loader   loader
			directive Directive
			reader    *os.File
		)

		if err != nil {
			return err
		}

		// ignore subdirectories.
		if info.IsDir() {
			if path != dir {
				l.Println("skipping directory", path)
				return filepath.SkipDir
			}

			return nil
		}

		ext := filepath.Ext(path)
		if _loader, found = extmap[ext]; !found {
			l.Println("no directive exists for", ext, ":", path, "skipping")
			return nil
		}

		if reader, err = os.Open(path); err != nil {
			return errors.Wrapf(err, "failed to open: %s", path)
		}
		defer reader.Close()

		if directive, err = _loader.Build(reader); err != nil {
			return errors.Wrapf(err, "failed to build directive for: %s", path)
		}

		results = append(results, directive)
		return nil
	})

	return results, err
}

func loaderToExts(logger logger, loaders ...loader) map[string]loader {
	m := make(map[string]loader, len(loaders))
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

type loader interface {
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
