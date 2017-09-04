package deployment

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func newLogger(root, prefix string) (dst *os.File, _ *log.Logger, err error) {
	if dst, err = os.OpenFile(filepath.Join(root, "deploy.log"), os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0600); err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return dst, log.New(io.MultiWriter(os.Stderr, dst), prefix, log.Flags()), nil
}

func logErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
