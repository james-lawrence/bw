package directives

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	Run(context.Context) error
}

// Load the directives from the provided directory.
func Load(l logger, dir string, loaders ...Loader) ([]Directive, error) {
	var (
		err error
	)

	results := make([]Directive, 0, 64)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		var (
			directive Directive
		)

		if err != nil {
			return err
		}

		// don't try to process a directory as a directive, instead
		// recurse into the directory.
		if info.IsDir() {
			return nil
		}

		if directive, err = infoToDirective(path, loaders...); err != nil {
			if skipError(err) {
				log.Println(err)
				return nil
			}

			return errors.Wrapf(err, "failed to build directive for: %s", path)
		}

		results = append(results, directive)
		return nil
	})

	// just return empty results if the directory did not exist.
	if os.IsNotExist(err) {
		err = nil
	}

	return results, err
}

func infoToDirective(path string, loaders ...Loader) (dir Directive, err error) {
	for _, l := range loaders {
		if dir, err = l.Load(path); err == nil {
			return dir, err
		}

		if skipError(err) {
			continue
		}

		return dir, err
	}

	return nil, skip{error: errors.Errorf("no directive exists for %s skipping", path)}
}

// LoadsExtensions convience func for use by directive factories to load files by extension.
func LoadsExtensions(name string, extensions ...string) error {
	aext := strings.TrimLeft(strings.ToLower(filepath.Ext(name)), ".")
	for _, ext := range extensions {
		// log.Println("comparing", ext, "==", aext)
		if strings.ToLower(ext) == aext {
			return nil
		}
	}

	return skip{error: errors.Errorf("%s did not match any %s", aext, extensions)}
}

func skipError(err error) bool {
	if _, ok := err.(invalid); ok {
		return true
	}

	return false
}

type invalid interface {
	invalid()
}

type skip struct {
	error
}

func (t skip) invalid() {}

// func loaderToExts(logger logger, loaders ...Loader) map[string]Loader {
// 	m := make(map[string]Loader, len(loaders))
// 	for _, l := range loaders {
// 		for _, ext := range l.Ext() {
// 			if _, found := m[ext]; found {
// 				logger.Println("extension is already mapped ignoring")
// 			} else {
// 				m[ext] = l
// 			}
// 		}
// 	}
//
// 	return m
// }

// // Loader represents a directive to be used for the specified extensions.
// type Loader interface {
// 	// extensions to match against.
// 	Ext() []string
// 	Build(io.Reader) (Directive, error)
// }

// Loader represents a directive factory which takse file information and builds
// directives from the file, or returns an error.
type Loader interface {
	Load(string) (Directive, error)
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
