package deployment

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/pkg/errors"
)

type dlog struct {
	uid string
	log *log.Logger
	dst *os.File
}

func (t dlog) Print(v ...interface{}) {
	t.log.Output(2, fmt.Sprintf("%s: %s", t.uid, fmt.Sprint(v...)))
}

func (t dlog) Printf(format string, v ...interface{}) {
	t.log.Output(2, fmt.Sprintf("%s: %s", t.uid, fmt.Sprintf(format, v...)))
}

func (t dlog) Println(v ...interface{}) {
	t.log.Output(2, fmt.Sprintf("%s: %s", t.uid, fmt.Sprintln(v...)))
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

	return dlog{dst: dst, uid: uid.String(), log: log.New(io.MultiWriter(os.Stderr, dst), prefix, log.Flags()^log.Lshortfile)}, nil
}

func StdErrLogger(prefix string) dlog {
	return dlog{uid: "", log: log.New(os.Stderr, prefix, log.Flags()^log.Lshortfile)}
}
