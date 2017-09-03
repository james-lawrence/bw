package fil

import (
  "os"
  "io"
  "io/ioutil"
)

func NopCloser(r io.Reader) io.ReadCloser {
  return ioutil.NopCloser(r)
}

func ReadAll(r io.Reader) ([]byte, error) {
  return ioutil.ReadAll(r)
}

func ReadDir(dirname string) ([]os.FileInfo, error) {
  return ioutil.ReadDir(dirname)
}

func ReadFile(filename string) ([]byte, error) {
  return ioutil.ReadFile(filename)
}

func TempDir(dir, prefix string) (name string, err error) {
  return ioutil.TempDir(dir, prefix)
}

func TempFile(dir, prefix string) (f *os.File, err error) {
  return ioutil.TempFile(dir, prefix)
}

func WriteFile(filename string, data []byte, perm os.FileMode) error {
  return ioutil.WriteFile(filename, data, perm)
}
