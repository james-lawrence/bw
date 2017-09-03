package fil

import (
  "os"
  "time"
  "os/exec"
)

func Getwd() (dir string, err error) {
  return os.Getwd()
}

func Chdir(dir string) error {
  return os.Chdir(dir)
}

func Chmod(name string, mode os.FileMode) error {
  return os.Chmod(name, mode)
}

func Readlink(name string) (string, error) {
  return os.Readlink(name)
}

func Chown(name string, uid, gid int) error {
  return os.Chown(name, uid, gid)
}

func Lchown(name string, uid, gid int) error {
  return os.Lchown(name, uid, gid)
}

func Chtimes(name string, atime time.Time, mtime time.Time) error {
  return os.Chtimes(name, atime, mtime)
}

func Mkdir(name string, perm os.FileMode) error {
  return os.Mkdir(name, perm)
}

func MkdirAll(path string, perm os.FileMode) error {
  return os.MkdirAll(path, perm)
}

func Remove(path string) error {
  return os.Remove(path)
}

func RemoveAll(path string) error {
  return os.RemoveAll(path)
}

func Rename(oldname, newname string) error {
  return os.Rename(oldname, newname)
}

func Symlink(oldname, newname string) error {
  return os.Symlink(oldname, newname)
}

func Link(oldname, newname string) error {
  return os.Link(oldname, newname)
}

func SameFile(fi1, fi2 os.FileInfo) bool {
  return os.SameFile(fi1, fi2)
}

func Truncate(name string, size int64) error {
  return os.Truncate(name, size)
}

func LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func Stat(name string) (os.FileInfo, error) {
  return os.Stat(name)
}

func Lstat(name string) (os.FileInfo, error) {
  return os.Lstat(name)
}
