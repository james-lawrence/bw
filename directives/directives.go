package directives

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie/directives/shell"
)

// Directive ...
type Directive interface {
	Run() error
}

// Load ...
func Load(dir string, loaders ...loader) ([]Directive, error) {
	var (
		err error
	)

	extmap := loaderToExts(loaders...)
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
				log.Println("skipping directory", path)
				return filepath.SkipDir
			}

			return nil
		}

		ext := filepath.Ext(path)
		if _loader, found = extmap[ext]; !found {
			log.Println("no directive exists for", ext, ":", path, "skipping")
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

func loaderToExts(loaders ...loader) map[string]loader {
	m := make(map[string]loader, len(loaders))
	for _, l := range loaders {
		for _, ext := range l.Ext() {
			if _, found := m[ext]; found {
				log.Println("extension is already mapped ignoring")
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

// ShellLoader directive.
type ShellLoader struct {
	Context shell.Context
}

// Ext extensions to succeed against.
func (ShellLoader) Ext() []string {
	return []string{".bwcmd"}
}

// Build builds a directive from the reader.
func (t ShellLoader) Build(r io.Reader) (Directive, error) {
	var (
		err  error
		cmds []shell.Exec
	)

	if cmds, err = shell.ParseYAML(r); err != nil {
		return nil, err
	}

	return closure(func() error {
		return shell.Execute(t.Context, cmds...)
	}), nil
}

type closure func() error

func (t closure) Run() error {
	return t()
}
