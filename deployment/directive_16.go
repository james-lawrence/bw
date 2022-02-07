//go:build go1.16
// +build go1.16

package deployment

import (
	"os"
)

func mkdirTemp(dir string, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}
