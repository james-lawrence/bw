package directives

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type logger interface {
	Output(depth int, msg string) error
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}

// Context global context for the agent.
type Context struct {
	Log           logger
	RootDirectory string
}

// Directive represents a computation to execute as part of the
// deployment.
type Directive interface {
	Run(context.Context) error
}

// Loaded a Directive with metadata about its original source.
type Loaded struct {
	Directive
	Path string
}

// Load the directives from the provided directory.
func Load(l logger, dir string, loaders ...Loader) ([]Loaded, error) {
	var (
		err error
	)

	extmap := loaderToExts(l, loaders...)
	results := make([]Loaded, 0, 64)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		var (
			found  bool
			loader Loader
			d      Directive
			reader *os.File
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

		if d, err = loader.Build(reader); err != nil {
			return errors.Wrapf(err, "failed to build directive for: %s", path)
		}

		results = append(results, Loaded{Directive: d, Path: path})
		return nil
	})

	// just return empty results if the directory did not exist.
	if os.IsNotExist(err) {
		err = nil
	}

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

type closure func(context.Context) error

func (t closure) Run(ctx context.Context) error {
	return t(ctx)
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
func (t NoopDirective) Run(ctx context.Context) error {
	return nil
}
