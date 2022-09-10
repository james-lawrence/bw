package protox

import (
	"os"

	"google.golang.org/protobuf/proto"
)

// WriteFile ...
func WriteFile(path string, perm os.FileMode, m proto.Message) (err error) {
	var (
		encoded []byte
	)

	if encoded, err = proto.Marshal(m); err != nil {
		return err
	}

	if err = os.WriteFile(path, encoded, perm); err != nil {
		return err
	}

	return nil
}

// ReadFile ...
func ReadFile(path string, m proto.Message) (err error) {
	var (
		encoded []byte
	)

	if encoded, err = os.ReadFile(path); err != nil {
		return err
	}

	return proto.Unmarshal(encoded, m)
}
