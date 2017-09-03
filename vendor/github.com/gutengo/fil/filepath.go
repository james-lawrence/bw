package fil

import (
  "os"
  "path/filepath"
)

const (
  Separator     = os.PathSeparator
  ListSeparator = os.PathListSeparator
)

func Abs(path string) (string, error) {
  return filepath.Abs(path)
}

func Base(path string) string {
  return filepath.Base(path)
}

func Clean(path string) string {
  return filepath.Clean(path)
}

func Dir(path string) string {
  return filepath.Dir(path)
}

func EvalSymlinks(path string) (string, error) {
  return filepath.EvalSymlinks(path)
}

func Ext(path string) string {
  return filepath.Ext(path)
}

func FromSlash(path string) string {
  return filepath.FromSlash(path)
}

func Glob(pattern string) (matches []string, err error) {
  return filepath.Glob(pattern)
}

func HasPrefix(p, prefix string) bool {
  return filepath.HasPrefix(p, prefix)
}

func IsAbs(path string) bool {
  return filepath.IsAbs(path)
}

func Join(elem ...string) string {
  return filepath.Join(elem...)
}

func Match(pattern, name string) (matched bool, err error) {
  return filepath.Match(pattern, name)
}

func Rel(basepath, targpath string) (string, error) {
  return filepath.Rel(basepath, targpath)
}

func Split(path string) (dir, file string) {
  return filepath.Split(path)
}

func SplitList(path string) []string {
  return filepath.SplitList(path)
}

func ToSlash(path string) string {
  return filepath.ToSlash(path)
}

func VolumeName(path string) (v string) {
  return filepath.VolumeName(path)
}

func Walk(root string, walkFn filepath.WalkFunc) error {
  return filepath.Walk(root, walkFn)
}
