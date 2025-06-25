package fsx

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// IsRegularFile returns true IFF a non-directory file exists at the provided path.
func IsRegularFile(path string) bool {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	if info.IsDir() {
		return false
	}

	return true
}

// MD5 computes digest of file contents.
// if something goes wrong logs and returns an empty string.
func MD5(path string) string {
	var (
		err  error
		read []byte
	)

	if read, err = os.ReadFile(path); err != nil {
		log.Println("digest failed", err)
		return ""
	}

	digest := md5.Sum(read)

	return hex.EncodeToString(digest[:])
}

// FileExists returns true IFF a non-directory file exists at the provided path.
func DirExists(path string) bool {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}

func CloneTree(ctx context.Context, dstdir string, rootdir string, archive fs.FS) (err error) {
	return fs.WalkDir(archive, rootdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Wrapf(err, "unable to walk directory: %s", path)
		}

		// allow clone tree to be cancellable.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() && rootdir == path {
			return nil
		}

		dst := filepath.Join(dstdir, strings.TrimPrefix(path, rootdir))
		if rootdir == path {
			dst = path
		}

		log.Println("cloning", rootdir, path, "->", dst, os.FileMode(0755), os.FileMode(0600))

		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}

		c, err := archive.Open(path)
		if err != nil {
			return err
		}
		defer c.Close()

		df, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer df.Close()

		if _, err := io.Copy(df, c); err != nil {
			return err
		}

		return nil
	})
}

func MkDirs(perm fs.FileMode, paths ...string) (err error) {
	for _, p := range paths {
		if err = os.MkdirAll(p, perm); err != nil {
			return errors.Wrapf(err, "unable to create directory: %s", p)
		}
	}

	return nil
}

func ErrIsNotExist(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

func IgnoreIsNotExist(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}
