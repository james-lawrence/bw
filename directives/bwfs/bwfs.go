package bwfs

import (
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"github.com/gutengo/fil"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/pkg/errors"
)

// Base-2 byte units.
const (
	KiB uint64 = 1024
	MiB        = KiB * 1024
	GiB        = MiB * 1024
	TiB        = GiB * 1024
	PiB        = TiB * 1024
	EiB        = PiB * 1024
)

// New ...
func New(l logger, root string) Executer {
	return Executer{
		log:  l,
		root: root,
	}
}

// Executer downloads and processes a set of archives.
// with a given context.
type Executer struct {
	log  logger
	root string
}

// Execute downloads and processes each archive.
func (t Executer) Execute(archives ...Archive) (err error) {
	for _, archive := range archives {
		if err = t.archive(archive); err != nil {
			return err
		}
	}

	return nil
}

func (t Executer) archive(a Archive) (err error) {
	var (
		info os.FileInfo
	)

	path := filepath.Join(t.root, a.URI)

	t.log.Println("archive", t.root, spew.Sdump(a))
	if info, err = os.Stat(path); err != nil {
		return errors.WithStack(err)
	}

	if info.IsDir() {
		return copyDirectory(path, a)
	}

	return copyArchiveFile(t.root, path, a)
}

func copyArchiveFile(root string, path string, a Archive) (err error) {
	var (
		dstp    string
		dstMode os.FileMode
	)

	dstMode = os.FileMode(a.Mode)
	dstp = a.Path
	if filepath.IsAbs(a.Path) {
		dstp = filepath.Join(root, a.Path)
	}

	if err = copyFile(path, dstp, dstMode); err != nil {
		return err
	}

	if err = chown(a); err != nil {
		return err
	}

	return nil
}

func copyFile(srcp, dstp string, dstMode os.FileMode) (err error) {
	var (
		src *os.File
		dst *os.File
		buf = make([]byte, 16*KiB)
	)

	debugx.Println("source path", srcp)
	debugx.Println("destination", dstp, dstMode)

	if src, err = os.Open(srcp); err != nil {
		return errors.WithStack(err)
	}
	defer src.Close()

	if dst, err = os.OpenFile(dstp, os.O_RDWR|os.O_CREATE|os.O_TRUNC, dstMode); err != nil {
		return errors.WithStack(err)
	}
	defer dst.Close()

	if _, err = io.CopyBuffer(dst, src, buf); err != nil {
		return errors.WithStack(err)
	}

	if err = os.Chmod(dstp, dstMode); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func copyDirectory(dir string, a Archive) (err error) {
	debugx.Println("copying directory", dir)
	walker := func(path string, info os.FileInfo, err error) error {
		var (
			dstp string
			mode = info.Mode()
		)

		if err != nil {
			return err
		}

		if dstp, err = filepath.Rel(dir, path); err != nil {
			return errors.WithStack(err)
		}

		dstp = filepath.Join(a.Path, dstp)
		debugx.Println("copying", path, "to", dstp)

		if info.IsDir() {
			return errors.WithStack(os.Mkdir(dstp, mode))
		}

		return copyFile(path, dstp, mode)
	}

	if err = filepath.Walk(dir, walker); err != nil {
		return errors.WithStack(err)
	}

	if err = chown(a); err != nil {
		return err
	}

	return errors.WithStack(os.Chmod(a.Path, os.FileMode(a.Mode)))
}

func chown(a Archive) (err error) {
	var (
		owner *user.User
		group *user.Group
		uid   int
		gid   int
	)

	if owner, err = user.Lookup(a.Owner); err != nil {
		return errors.WithStack(err)
	}

	if group, err = user.LookupGroup(a.Group); err != nil {
		return errors.WithStack(err)
	}

	if uid, err = strconv.Atoi(owner.Uid); err != nil {
		return errors.WithStack(err)
	}

	if gid, err = strconv.Atoi(group.Gid); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(fil.ChownR(a.Path, uid, gid))
}

func printIfErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

type logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}
