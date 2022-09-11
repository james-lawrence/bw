package deployment

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
)

type dlog struct {
	*log.Logger
	uid string
	dst *os.File
}

func (t dlog) Write(b []byte) (n int, err error) {
	return t.Logger.Writer().Write(b)
}

func (t dlog) Close() error {
	if t.dst != nil {
		return errorsx.Compact(t.dst.Sync(), t.dst.Close())
	}

	return nil
}

func newLogger(uid bw.RandomID, root, prefix string) (_dlog dlog, err error) {
	var (
		dst *os.File
	)

	if dst, err = os.OpenFile(filepath.Join(root, bw.DeployLog), os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0600); err != nil {
		return _dlog, errors.WithStack(err)
	}

	return dlog{dst: dst, uid: uid.String(), Logger: log.New(io.MultiWriter(os.Stderr, dst), prefix, log.Flags()^log.Lshortfile)}, nil
}

// StdErrLogger ...
func StdErrLogger(prefix string) dlog {
	return dlog{uid: "", Logger: log.New(os.Stderr, prefix, log.Flags()^log.Lshortfile^log.Ldate^log.Ltime)}
}
