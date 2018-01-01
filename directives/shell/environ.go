package shell

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

// EnvironFromFile loads an environment from a file.
func EnvironFromFile(path string) (environ []string, err error) {
	var (
		src *os.File
	)

	if src, err = os.Open(path); err != nil {
		return environ, errors.WithStack(err)
	}
	defer src.Close()

	return EnvironFromReader(src)
}

// EnvironFromReader loads an environment from a reader.
func EnvironFromReader(r io.Reader) (environ []string, err error) {
	var (
		raw []byte
	)

	if raw, err = ioutil.ReadAll(r); err != nil {
		return environ, errors.WithStack(err)
	}

	return Environ(string(raw))
}

// Environ loads an environment from a string.
func Environ(s string) (environ []string, err error) {
	var (
		ir map[string]string
	)
	if ir, err = godotenv.Unmarshal(string(s)); err != nil {
		return environ, errors.WithStack(err)
	}

	environ = make([]string, 0, len(ir))
	for k, v := range ir {
		var line string
		if line, err = godotenv.Marshal(map[string]string{k: v}); err != nil {
			return environ, errors.WithStack(err)
		}

		environ = append(environ, line)
	}

	return environ, nil
}

// MustEnviron panics if err is not nil.
func MustEnviron(environ []string, err error) []string {
	if err != nil {
		panic(err)
	}

	return environ
}
