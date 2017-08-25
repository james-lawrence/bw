package bw

import (
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v1"

	"github.com/pkg/errors"
)

// ExpandAndDecodeFile ...
func ExpandAndDecodeFile(path string, dst interface{}) (err error) {
	var (
		raw []byte
	)

	if _, err = os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	if raw, err = ioutil.ReadFile(path); err != nil {
		return errors.WithStack(err)
	}

	return ExpandAndDecode(raw, dst)
}

// ExpandAndDecode expands environment variables within the file at the specified
// path and then decodes it as yaml.
func ExpandAndDecode(raw []byte, dst interface{}) (err error) {
	return ExpandEnvironAndDecode(raw, dst, os.Getenv)
}

// ExpandEnvironAndDecode ...
func ExpandEnvironAndDecode(raw []byte, dst interface{}, mapping func(string) string) (err error) {
	return yaml.Unmarshal([]byte(os.Expand(string(raw), mapping)), dst)
}
