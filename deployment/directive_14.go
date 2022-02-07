//go:build !go1.16
// +build !go1.16

package deployment

import (
	"os"
	"path/filepath"
)

func mkdirTemp(dir string, pattern string) (string, error) {
	return os.Mkdir(filepath.Join(dir, ".bw-tmp"), 0700)
}
