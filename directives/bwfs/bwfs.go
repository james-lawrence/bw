package bwfs

import (
	"log"
	"os"
	"os/user"
	"strconv"

	"github.com/gutengo/fil"
	"github.com/james-lawrence/bw/inflaters"
	"github.com/james-lawrence/bw/storage"
	"github.com/pkg/errors"
)

type downloader interface {
	New(string) storage.Downloader
}

type downloaderClosure func(string) storage.Downloader

func (t downloaderClosure) New(s string) storage.Downloader {
	return t(s)
}

type inflater interface {
	New(location, destination string, perm os.FileMode) inflaters.Inflater
}

type inflaterClosure func(location, destination string, perm os.FileMode) inflaters.Inflater

func (t inflaterClosure) New(location, destination string, perm os.FileMode) inflaters.Inflater {
	return t(location, destination, perm)
}

// New ...
func New() Executer {
	return Executer{
		downloader: storage.New(),
		inflater:   inflaterClosure(inflaters.New),
	}
}

// Executer downloads and processes a set of archives.
// with a given context.
type Executer struct {
	downloader downloader
	inflater   inflater
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
	local := t.downloader.New(a.URI).Download()
	defer local.Close()
	log.Println(a)

	if err = t.inflater.New(a.URI, a.Path, os.FileMode(a.Mode)).Inflate(local); err != nil {
		return err
	}

	if err = t.chown(a); err != nil {
		return err
	}

	return nil
}

func (t Executer) chown(a Archive) (err error) {
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
