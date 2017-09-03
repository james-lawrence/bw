package fil

import (
  "os"
  "path/filepath"
  "os/exec"
	"errors"
	"io"
	"io/ioutil"
)

type Options struct {
	Recursive       bool
	PreserveLinks   bool
	PreserveModTime bool
}

func ChmodR(name string, mode os.FileMode) error {
	return filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chmod(path, mode)
		}
		return err
	})
}

func ChownR(path string, uid, gid int) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, uid, gid)
		}
		return err
	})
}

func MkdirP(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func Mv(oldname, newname string) error {
	return os.Rename(oldname, newname)
}

func Rm(name string) error {
	return os.Remove(name)
}

func RmRF(path string) error {
	return os.RemoveAll(path)
}

func Which(file string) (string, error) {
	return exec.LookPath(file)
}

func Cp(src, dest string) error {
  return CpWithOptions(src, dest, Options{PreserveLinks: true})
}

func CpR(source, dest string) error {
	return CpWithOptions(source, dest, Options{Recursive: true, PreserveLinks: true})
}

func CpSymlinkContent(src, dest string) error {
  return CpWithOptions(src, dest, Options{PreserveLinks: false})
}

func CpDirOnly(src, dest string) error {
  fi, err := os.Lstat(src)
  if err != nil {
    return err
  }
  if !fi.IsDir() {
    return errors.New("source is not a directory -- "+src)
  }
  if _, err := os.Open(dest); !os.IsNotExist(err) {
    return errors.New("destination already exists -- "+dest)
  }
  if err := os.MkdirAll(dest, fi.Mode().Perm()); err != nil {
    return err
  }
  return nil
}

func CpWithOptions(source, dest string, args Options) (err error) {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return
	}

	if sourceInfo.IsDir() {
		// Handle the dir case
		if !args.Recursive {
			return errors.New("source is a directory")
		}

		// ensure dest dir does not already exist
    if _, err := os.Open(dest); !os.IsNotExist(err) {
			return errors.New("destination already exists")
		}

		// create dest dir
		if err = os.MkdirAll(dest, sourceInfo.Mode()); err != nil {
			return
		}

		files, err := ioutil.ReadDir(source)
		if err != nil {
			return err
		}

		for _, file := range files {
			if err = CpWithOptions(source+"/"+file.Name(), dest+"/"+file.Name(), args); err != nil {
				return err
			}
		}
	} else {
		// Handle the file case
		si, err := os.Lstat(source)
		if err != nil {
			return err
		}

		if args.PreserveLinks && !si.Mode().IsRegular() {
			return cpSymlink(source, dest)
		}

		//open source
		in, err := os.Open(source)
		if err != nil {
			return err
		}
		defer in.Close()

		//create dest
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer func() {
			cerr := out.Close()
			if err == nil {
				err = cerr
			}
		}()

		//copy to dest from source
		if _, err = io.Copy(out, in); err != nil {
			return err
		}

		if err = out.Chmod(si.Mode()); err != nil {
			return err
		}

		if args.PreserveModTime {
			if err = os.Chtimes(dest, si.ModTime(), si.ModTime()); err != nil {
				return err
			}
		}

		//sync dest to disk
		err = out.Sync()
	}

	return
}

func cpSymlink(src, dest string) error {
  linkTarget, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(linkTarget, dest)
}
