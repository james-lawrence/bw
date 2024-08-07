package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Pack ...
func Pack(dst io.Writer, paths ...string) (err error) {
	var (
		gw *gzip.Writer
		tw *tar.Writer
	)

	gw = gzip.NewWriter(dst)
	defer gw.Close()
	tw = tar.NewWriter(gw)
	defer tw.Close()

	for _, basepath := range paths {
		walker := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// skip the root directory itself.
			if basepath == path && info.IsDir() {
				return nil
			}

			return write(basepath, path, tw, info)
		}

		if err = filepath.Walk(basepath, walker); err != nil {
			return err
		}
	}

	return errors.Wrap(tw.Flush(), "failed to flush archive")
}

// Unpack unpacks the archive from the reader into the root directory.
func Unpack(root string, r io.Reader) (err error) {
	var (
		dst *os.File
		gzr *gzip.Reader
		tr  *tar.Reader
	)

	if gzr, err = gzip.NewReader(r); err != nil {
		return errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzr.Close()

	tr = tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil
		// return any other error
		case err != nil:
			return err
		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(root, header.Name)

		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err = os.Stat(target); os.IsNotExist(err) {
				if err = os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return errors.Wrapf(err, "failed to create directory: %s", target)
				}
			} else if err != nil {
				return errors.Wrapf(err, "failed to stat directory: %s", target)
			}
		// if it's a file create it
		case tar.TypeReg:
			writefile := func() error {
				if dst, err = os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode)); err != nil {
					return errors.Wrapf(err, "failed to open file: %s", target)
				}
				defer dst.Close()

				// copy over contents
				if _, err = io.Copy(dst, tr); err != nil {
					return errors.Wrapf(err, "failed to copy contents: %s", target)
				}

				return nil
			}

			if err := writefile(); err != nil {
				return err
			}
		}
	}
}

func write(basepath, path string, tw *tar.Writer, info os.FileInfo) (err error) {
	var (
		src    *os.File
		header *tar.Header
		target string
	)

	if target, err = filepath.Rel(basepath, path); err != nil {
		return errors.Wrapf(err, "failed to compute path: %s", path)
	}

	if src, err = os.Open(path); err != nil {
		return errors.Wrap(err, "failed to open path")
	}
	defer src.Close()

	if header, err = tar.FileInfoHeader(info, path); err != nil {
		return errors.Wrap(err, "failed to created header")
	}
	header.Name = target

	if err = tw.WriteHeader(header); err != nil {
		return errors.Wrapf(err, "failed to write header to tar archive: %s", path)
	}

	// return on directories since there will be no content to tar
	if info.Mode().IsDir() {
		return nil
	}

	if _, err = io.Copy(tw, src); err != nil {
		return errors.Wrapf(err, "failed to write contexts to tar archive: %s", path)
	}

	return nil
}
