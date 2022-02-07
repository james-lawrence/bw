//go:build !go1.16
// +build !go1.16

package deployment

import (
	"os"
	"path/filepath"
)

func mkdirTemp(dir string, pattern string) (string, error) {
	p := filepath.Join(dir, ".bw-tmp")
	return p, os.Mkdir(p, 0700)
}
