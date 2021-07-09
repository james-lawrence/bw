// +build go1.16

package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

func (t *workspaceCreate) generate(ctx *kingpin.ParseContext) (err error) {
	if err = errors.WithStack(os.MkdirAll(t.path, 0755)); err != nil {
		return err
	}

	var (
		root          = ".assets/workspace/empty"
		archive fs.FS = workspaceempty
	)

	if t.includeExamples {
		root = ".assets/workspace/example1"
		archive = workspaceexample1
	}

	return fs.WalkDir(archive, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		dst := filepath.Join(t.path, strings.TrimPrefix(path, root))

		log.Println("cloning", root, path, "->", dst, os.FileMode(0755), os.FileMode(0600))

		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}

		c, err := archive.Open(path)
		if err != nil {
			return err
		}

		df, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}

		if _, err := io.Copy(df, c); err != nil {
			return err
		}

		return nil
	})
}

//go:embed .assets/workspace/empty/.local/.gitignore .assets/workspace/empty/.remote/.gitignore
var workspaceempty embed.FS

//go:embed .assets/workspace/example1/*
var workspaceexample1 embed.FS
